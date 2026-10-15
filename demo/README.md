# demo: sal on Cloud Run for knowledge-and-insights

Terraform for running [sal](https://github.com/cgs-earth/sal) against an existing SAL project
that harvests ~10 Geoconnex sitemaps through `geoconnex-salmodule` and serves the result with
`sal serve --with-ui` on Cloud Run.

## Architecture

```
 manual trigger                 Cloud Build (cloudbuild.yaml)
 ───────────────►   ┌─────────────────────────────────────────────────────────┐
                     │ 1. Harvest: git clone your SAL project, `sal build`     │
                     │    (runs geoconnex-salmodule's container on the docker  │
                     │    daemon Cloud Build steps get for free)               │
                     │ 2. Bake: docker build an image with that already-built  │
                     │    project (Iceberg table included) baked in → push to │
                     │    Artifact Registry                                    │
                     │ 3. Deploy: `gcloud run deploy` with the new image       │
                     └─────────────────────────────────────────────────────────┘
                                                                  │
                                                                  ▼
                                              Cloud Run (scale to 0, public HTTPS)
                                              `sal serve --with-ui` — nothing to
                                              fetch on cold start, it's all in the image
```

Why baked into the image rather than uploaded somewhere `sal serve` reads at request time: `sal
serve` always reads a local `.sal/data` directory next to wherever it's run — it never points
`iceberg_scan()` at a remote bucket on its own. There's deliberately no `sal upload` step and no
GCS bucket here: uploading a second copy of the Iceberg table that nothing ever queries would
just be a standing storage cost with no consumer. (`sal upload` exists for the case where some
*other* DuckDB wants to run `iceberg_scan('gs://...')` directly — add it back if you ever have
that consumer.) Baking the build into the image instead means a cold start on a scale-to-zero
service has zero network round trips before it can answer a query.

This is the same two-build-steps-plus-deploy pattern [sal's own demo
deployment](https://github.com/cgs-earth/sal/blob/main/cloudbuild.yaml) uses, adapted to build
your real project instead of their sample data.

## What Terraform manages

- Artifact Registry Docker repository (the baked serve image)
- Two service accounts: `sal-builder` (runs the pipeline below) and `sal-run` (the Cloud Run
  runtime identity, granted nothing beyond the defaults since the image is self-contained).
  `roles/iam.serviceAccountUser` on `sal-builder` is granted to whichever identity ran
  `terraform apply` (via a `google_client_openid_userinfo` lookup), since being able to use
  `sal-builder`'s own permissions is separate from `sal-builder` having them — if someone else
  needs to run `gcloud builds submit`, add them with the `builder_impersonators` variable.
- The Cloud Run service shell, created with a placeholder image and `min_instance_count = 0`.
  Terraform never manages the *running* image after the first deploy (see the `lifecycle`
  block on `google_cloud_run_v2_service.sal`) — `cloudbuild.yaml`'s deploy step owns that.
- `roles/run.invoker` for `allUsers`, since you asked for public HTTPS access. Swap this for an
  IAP/allowlist binding if that changes.

**Scale to zero**: `min_instance_count = 0` in Terraform sets it on the very first (placeholder)
revision, but every later revision comes from an imperative `gcloud run deploy` in
`cloudbuild.yaml`, which by default inherits unspecified settings from the previous revision
rather than Terraform's state. To not depend on that inheritance, the Deploy step passes
`--min-instances=0` explicitly on every single deploy — so the service can never end up
accidentally pinned to a warm instance, no matter how many times it's redeployed or who runs the
`gcloud` command by hand afterwards. CPU idle-throttling (what actually lets an instance cost
nothing between requests) is also left at its default, which is already scale-to-zero-friendly;
nothing here turns on "CPU always allocated".

What it deliberately does **not** manage: any schedule or trigger. You asked for harvest-on-demand
only, so refreshing the data is always a manual `gcloud builds submit` (below), never automatic.

## Deploying

Every variable in `variables.tf` has a default — including `project_id`, which defaults to
`"knowledge-and-insights"` — so there's no tfvars file to create. Double-check that default is
actually your project's real ID (GCP project IDs can differ from a project's display name) before
the first apply; if it isn't, override it without a file:

```sh
make apply TF_APPLY_FLAGS='-var project_id=your-real-project-id'
```

`Makefile` wraps the `terraform apply` + `gcloud builds submit` flow below into one command, so
you don't have to copy outputs between them by hand. It needs `terraform` and `gcloud` on `PATH`
and `gcloud` already pointed at the right account/project.

```sh
make deploy SAL_PROJECT_REPO=https://github.com/you/your-sal-project
```

That's `terraform apply` (creates the Artifact Registry repo, service accounts, and the Cloud Run
service shell — the Cloud Run service starts out running a Google placeholder image until the
build below replaces it) followed by the harvest/bake/deploy pipeline, then it prints
the Cloud Run URL. `SAL_PROJECT_REPO` is the existing repo with the task instance that lists your
~10 sitemap ids (see `example-task.ttl` in this directory for what that declaration looks like) —
not managed by Terraform, since it's your repo. `make help` lists the individual steps
(`apply`, `build`, `plan`, `url`, `destroy`) if you want to run just one of them — `make build
SAL_PROJECT_REPO=...` on its own re-harvests and redeploys without touching the infrastructure,
and is what you run by hand whenever you want fresher data; nothing here triggers it
automatically. A build takes a few minutes depending on how large your chosen sitemaps are — the
10 in `example-task.ttl` total well under a gigabyte, but swap in `ref:hu12` or similar and expect
it to take a lot longer (that one alone is multiple gigabytes).

Prefer to run the steps yourself instead of through `make`? `terraform output build_command`
prints the equivalent `gcloud builds submit` command with every substitution except
`_SAL_PROJECT_REPO` already filled in — fill that in and run it from the root of this
geoconnex-salmodule checkout (not from `demo/`; the build context needs `demo/Dockerfile`).

Either way, once it finishes:

```sh
make url   # or: terraform output cloud_run_url   # the public sal serve UI — SPARQL, SQL, map, stats tabs
```

## Caveats worth knowing before you rely on this

- **Private SAL project repo**: `git clone` in the Harvest step assumes a public HTTPS URL. If
  your existing repo is private, the Harvest step needs credentials (a deploy key or token) wired
  in as a Cloud Build secret — not set up here.
- **`sal build` needs `.sal/config.jsonld` already committed** in your project repo (i.e. `sal
  init` has already been run there at some point). A repo that was never initialized will fail
  clearly rather than silently.
- **Build path**: `_SAL_BUILD_PATH` defaults to `data/`, matching the convention in the `sal`
  skill's examples. Override it in `--substitutions` if your repo's RDF lives somewhere else.
- **Not load-tested**: `sal`'s own demo deployment is the only other real-world reference for
  this exact pattern I could ground this against; I haven't run this specific pipeline
  end-to-end against GCP myself. Treat the first `gcloud builds submit` as the real test, and
  check the Cloud Build logs if a step behaves unexpectedly.
