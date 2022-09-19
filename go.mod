module github.com/calyptia/cli

go 1.19

replace github.com/calyptia/cli/k8s => ./k8s

require (
	code.cloudfoundry.org/bytefmt v0.0.0-20211005130812-5bb3c17173e5
	github.com/Masterminds/sprig/v3 v3.2.2
	github.com/alecthomas/assert/v2 v2.1.0
	github.com/aws/aws-sdk-go-v2 v1.16.15
	github.com/aws/aws-sdk-go-v2/config v1.17.5
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.54.4
	github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi v1.13.17
	github.com/calyptia/api v0.9.0
	github.com/calyptia/cli/k8s v0.0.0-00010101000000-000000000000
	github.com/calyptia/core-images-index/go-index v0.0.0-20220907103121-f0a8df68bedc
	github.com/calyptia/go-bubble-table v0.2.1
	github.com/calyptia/go-fluentbit-config v0.3.2
	github.com/charmbracelet/bubbletea v0.22.1
	github.com/charmbracelet/lipgloss v0.6.0
	github.com/hako/durafmt v0.0.0-20210608085754-5c1018a4e16b
	github.com/joho/godotenv v1.4.0
	github.com/matryer/moq v0.2.7
	github.com/pkg/errors v0.9.1
	github.com/sethvargo/go-retry v0.2.3
	github.com/spf13/cobra v1.5.0
	github.com/zalando/go-keyring v0.2.1
	golang.org/x/sync v0.0.0-20220819030929-7fc1605a5dde
	golang.org/x/term v0.0.0-20220722155259-a9ba230a4035
	google.golang.org/api v0.95.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.25.0
	k8s.io/apimachinery v0.25.0
	k8s.io/client-go v0.25.0
)

require (
	cloud.google.com/go/compute v1.9.0 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.1.1 // indirect
	github.com/alecthomas/participle/v2 v2.0.0-alpha9 // indirect
	github.com/alecthomas/repr v0.1.0 // indirect
	github.com/alessio/shellescape v1.4.1 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.12.18 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.12.15 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.15 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.22 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.11.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.13.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.16.17 // indirect
	github.com/aws/smithy-go v1.13.3 // indirect
	github.com/charmbracelet/bubbles v0.13.0 // indirect
	github.com/containerd/console v1.0.3 // indirect
	github.com/danieljoos/wincred v1.1.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emicklei/go-restful/v3 v3.9.0 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.20.0 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/gnostic v0.6.9 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.1.0 // indirect
	github.com/googleapis/gax-go/v2 v2.5.1 // indirect
	github.com/hashicorp/go-version v1.6.0 // indirect
	github.com/hexops/gotextdiff v1.0.3 // indirect
	github.com/huandu/xstrings v1.3.1 // indirect
	github.com/imdario/mergo v0.3.13 // indirect
	github.com/inconshreveable/mousetrap v1.0.1 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/juju/ansiterm v1.0.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/lunixbochs/vtclean v1.0.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-isatty v0.0.16 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/muesli/ansi v0.0.0-20211031195517-c9f0611b6c70 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/reflow v0.3.0 // indirect
	github.com/muesli/termenv v0.12.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/peterhellberg/link v1.1.0 // indirect
	github.com/rivo/uniseg v0.3.4 // indirect
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	go.opencensus.io v0.23.0 // indirect
	golang.org/x/crypto v0.0.0-20220427172511-eb4f295cb31f // indirect
	golang.org/x/mod v0.6.0-dev.0.20220106191415-9b9b3d81d5e3 // indirect
	golang.org/x/net v0.0.0-20220826154423-83b083e8dc8b // indirect
	golang.org/x/oauth2 v0.0.0-20220822191816-0ebed06d0094 // indirect
	golang.org/x/sys v0.0.0-20220829200755-d48e67d00261 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20220722155302-e5dcc9cfc0b9 // indirect
	golang.org/x/tools v0.1.10 // indirect
	golang.org/x/xerrors v0.0.0-20220609144429-65e65417b02f // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20220902135211-223410557253 // indirect
	google.golang.org/grpc v1.49.0 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/klog/v2 v2.80.0 // indirect
	k8s.io/kube-openapi v0.0.0-20220803164354-a70c9af30aea // indirect
	k8s.io/utils v0.0.0-20220823124924-e9cbc92d1a73 // indirect
	sigs.k8s.io/json v0.0.0-20220713155537-f223a00ba0e2 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)
