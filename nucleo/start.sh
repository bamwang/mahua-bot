mkdir -p $GOPATH/src/github.com/bamwang/
rm -fr $GOPATH/src/github.com/bamwang/mahua-bot
mv $GOPATH/src/app $GOPATH/src/github.com/bamwang/mahua-bot
cd $GOPATH/src/github.com/bamwang/mahua-bot
./start.sh