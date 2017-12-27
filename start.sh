go build
if [ -f env.rc ]; then
  . env.rc
  echo "env.rc loaded"
fi
./mahua-bot