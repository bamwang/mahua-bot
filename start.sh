go build
if [ -f env.rc ]; then
  set -o allexport
  source env.rc
  set +o allexport
  echo "env.rc loaded"
fi
./mahua-bot