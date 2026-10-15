# Built by demo/cloudbuild.yaml's "Bake serve image" step, with a build
# context of /workspace (the whole Cloud Build checkout, not just demo/),
# because it copies in /workspace/project — the SAL project the "Harvest"
# step just ran `sal build` on, Iceberg table and all. Baking the already
# -built project in means a Cloud Run cold start never has to reach GCS,
# Docker, or GitHub: everything `sal serve` needs is already on disk.
FROM ghcr.io/cgs-earth/sal:latest

WORKDIR /app/project
COPY project/ ./

# The base image's ENTRYPOINT is already ["/app/sal"]; `sal serve` always
# listens on :8080 and ignores $PORT, which happens to match the Cloud Run
# default container port, so no port wiring is needed here.
CMD ["serve", "--with-ui"]
