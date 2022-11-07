basicCoverageRate=700
repositoryName="go-amazing"
coverageResult=$(go tool cover -func=coverage.out)

echo "$coverageResult\n-----------------------"

coverageRate=$(go tool cover -func=coverage.out | grep "(statements" | sed 's/[^0-9]*//g');

if [ $coverageRate -gt $basicCoverageRate ]
then
  echo "✅   test coverage rate is higher than 70%"
  exit 0 
else
  echo "⚠️   test coverage rate should higher than 70%"
  exit 1
fi