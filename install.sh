#!/usr/bin/env sh
set -ef

if [ -n "${DEBUG}" ]; then
  set -x
fi

_sudo() {
  [ "$(id -u)" -eq 0 ] || set -- command sudo "$@"
  "$@"
}

_detect_arch() {
  case $(uname -m) in
    amd64 | x86_64)
      echo "amd64"
      ;;
    arm64 | aarch64)
      echo "arm64"
      ;;
    i386)
      echo "i386"
      ;;
    *)
      echo "Unsupported processor architecture"
      return 1
      ;;
  esac
}

_detect_os() {
  case $(uname) in
    Linux)
      echo "linux"
      ;;
    Darwin)
      echo "darwin"
      ;;
    Windows)
      echo "windows"
      ;;
  esac
}

_download_url() {
  local arch="$(_detect_arch)"
  local os="$(_detect_os)"
  local version=$cli_VERSION

  if [ -z "$cli_VERSION" ]; then
    version=$(curl -s https://api.github.com/repos/calyptia/cli/releases/latest 2> /dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
  fi

  local trailedVersion=$(echo $version | tr -d v)
  echo "https://github.com/calyptia/cli/releases/download/${version}/cli_${trailedVersion}_${os}_${arch}.tar.gz"
}

echo "Downloading Calyptia CLI from URL: $(_download_url)"
curl --progress-bar --output cli.tar.gz -SLf "$(_download_url)"
tar -xzf cli.tar.gz calyptia
rm cli.tar.gz

install_dir=$1
if [ "$install_dir" != "" ]; then
  mkdir -p "$install_dir"
  mv calyptia "${install_dir}/calyptia"
  echo "Calyptia CLI installed in ${install_dir}"
  exit 0
fi

if [ "$(id -u)" -ne 0 ]; then
  echo "Sudo rights are needed to move the binary to /usr/local/bin, please type your password when asked"
  _sudo mv calyptia /usr/local/bin/calyptia
else
  mv calyptia /usr/local/bin/calyptia
fi

echo "Calyptia CLI installed in /usr/local/bin/calyptia"