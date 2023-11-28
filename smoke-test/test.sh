echo "getting latest operator image version"
latest=$(curl -sf https://api.github.com/repos/calyptia/core-operator-releases/releases | jq -r .[0].tag_name)

echo "getting version before latest operator image version"
beforelatest=$(curl -sf https://api.github.com/repos/calyptia/core-operator-releases/releases | jq -r .[1].tag_name)
echo "test install operator into default namespace"
./calyptia install operator 
./calyptia update operator --version $beforelatest --verbose
./calyptia update operator --version $latest --verbose
./calyptia uninstall operator 

echo "test install operator into test namespace"
./calyptia install operator --kube-namespace test
./calyptia update operator --version $beforelatest --kube-namespace test--verbose


