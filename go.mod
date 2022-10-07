module github.com/calyptia/cli

go 1.19

replace github.com/calyptia/cli/k8s => ./k8s

require (
	code.cloudfoundry.org/bytefmt v0.0.0-20211005130812-5bb3c17173e5
	github.com/Masterminds/sprig/v3 v3.2.2
	github.com/alecthomas/assert/v2 v2.1.0
	github.com/aws/aws-sdk-go-v2 v1.16.16
	github.com/aws/aws-sdk-go-v2/config v1.17.7
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.58.0
	github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi v1.13.19
	github.com/calyptia/api v0.14.0
	github.com/calyptia/cli/k8s v0.0.0-00010101000000-000000000000
	github.com/calyptia/core-images-index/go-index v0.0.0-20220920114028-a0b0d274f5d2
	github.com/calyptia/go-bubble-table v0.2.1
	github.com/calyptia/go-fluentbit-config v0.4.1
	github.com/charmbracelet/bubbletea v0.22.1
	github.com/charmbracelet/lipgloss v0.6.0
	github.com/go-logfmt/logfmt v0.5.1
	github.com/hako/durafmt v0.0.0-20210608085754-5c1018a4e16b
	github.com/joho/godotenv v1.4.0
	github.com/matryer/moq v0.2.7
	github.com/pkg/errors v0.9.1
	github.com/sethvargo/go-retry v0.2.3
	github.com/spf13/cobra v1.5.0
	github.com/zalando/go-keyring v0.2.1
	golang.org/x/exp v0.0.0-20220921164117-439092de6870
	golang.org/x/sync v0.0.0-20220907140024-f12130a52804
	golang.org/x/term v0.0.0-20220919170432-7a66f970e087
	golang.org/x/time v0.0.0-20220920022843-2ce7c2934d45
	google.golang.org/api v0.96.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.25.1
	k8s.io/apimachinery v0.25.1
	k8s.io/client-go v0.25.1
)

require (
	cloud.google.com/go/compute v1.10.0 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.1.1 // indirect
	github.com/alecthomas/participle/v2 v2.0.0-alpha9 // indirect
	github.com/alecthomas/repr v0.1.0 // indirect
	github.com/alessio/shellescape v1.4.1 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.12.20 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.12.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.24 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.11.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.13.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.16.19 // indirect
	github.com/aws/smithy-go v1.13.3 // indirect
	github.com/charmbracelet/bubbles v0.14.0 // indirect
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
	github.com/huandu/xstrings v1.3.2 // indirect
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
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/muesli/ansi v0.0.0-20211031195517-c9f0611b6c70 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/reflow v0.3.0 // indirect
	github.com/muesli/termenv v0.12.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/peterhellberg/link v1.1.0 // indirect
	github.com/rivo/uniseg v0.4.2 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	go.opencensus.io v0.23.0 // indirect
	golang.org/x/crypto v0.0.0-20220919173607-35f4265a4bc0 // indirect
	golang.org/x/mod v0.6.0-dev.0.20220419223038-86c51ed26bb4 // indirect
	golang.org/x/net v0.0.0-20220921155015-db77216a4ee9 // indirect
	golang.org/x/oauth2 v0.0.0-20220909003341-f21342109be1 // indirect
	golang.org/x/sys v0.0.0-20220919091848-fb04ddd9f9c8 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/tools v0.1.12 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20220920201722-2b89144ce006 // indirect
	google.golang.org/grpc v1.49.0 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/klog/v2 v2.80.1 // indirect
	k8s.io/kube-openapi v0.0.0-20220803164354-a70c9af30aea // indirect
	k8s.io/utils v0.0.0-20220823124924-e9cbc92d1a73 // indirect
	sigs.k8s.io/json v0.0.0-20220713155537-f223a00ba0e2 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)
