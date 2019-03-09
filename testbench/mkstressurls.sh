# creates a list of urls that a benchmark utility can hit

SCRIPTDIR=$(dirname "$0")

rm $SCRIPTDIR/stress-urls.txt

for i in {1..100}
do
	echo "http://as$i.teleclimber.dropserver.develop:3000/" >> $SCRIPTDIR/stress-urls.txt
done