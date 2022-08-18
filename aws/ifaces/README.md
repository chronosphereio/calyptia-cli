
Unfortunately the V2 of aws go client, doesn't provide interface types
for properly mock the client side. 

```shell
ifacemaker -f '/Users/niedbalski/go/src/github.com/calyptia/cli/vendor/github.com/aws/aws-sdk-go-v2/service/ec2/*.go' -s Client -i Client -p mocks -m github.com/aws/aws-sdk-go-v2/service/ec2 -o ./ec2_client_interface.go
```
Then the moq client has been generated with:

```shell
moq -out ec2_client_mock.go . Client
```