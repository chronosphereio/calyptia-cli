#!/usr/bin/env sh
set -ef

# Use CLI_INSTALL_DIR or first parameter to specify where to install the Calyptia CLI, otherwise it will default to /usr/local/bin.
# Use CLI_VERSION to specify the version to install, otherwise it will default to latest.
# Use CLI_DOWNLOAD_OUTPUT_DIR to specify another directory other than the current one ($PWD) to download artefacts.
# Use CLI_ARTEFACT_PREFIX to override the name of the artefact used when downloading.
# Use CLI_DOWNLOAD_REPO to override the source repository to take the artefacts from.
# Use CLOUD_API_URL to override the default Cloud API instance to use, this will unset the token as well as a precaution.

if [ -n "${DEBUG}" ]; then
  set -x
fi

# Specify a specific directory to install into
install_dir="${CLI_INSTALL_DIR:-$1}"
if [ -n "$install_dir" ]; then
  mkdir -p "$install_dir"
else
  install_dir="/usr/local/bin"
fi

_download_output_dir="${CLI_DOWNLOAD_OUTPUT_DIR:-$PWD}"
if [  "$_download_output_dir" != "$PWD" ]; then
  mkdir -p "$_download_output_dir"
fi
if [ -z "$_download_output_dir" ]; then
  _download_output_dir="."
fi

_download_repo=${CLI_DOWNLOAD_REPO:-chronosphereio/calyptia-cli}

CLOUD_API_URL=${CLOUD_API_URL:-}

# Cope with legacy names
CLI_VERSION=${CLI_VERSION:-}
if [ -n "${cli_VERSION:-}" ]; then
  CLI_VERSION=$cli_VERSION
fi

CLI_ARTEFACT_PREFIX=${CLI_ARTEFACT_PREFIX:-calyptia-cli}
if [ -n "${cli_ARTEFACT_PREFIX:-}" ]; then
  CLI_ARTEFACT_PREFIX=$cli_ARTEFACT_PREFIX
fi

if ! command -v curl > /dev/null 2>&1; then
  echo "ERROR: missing curl command, please install"
  exit 1
fi
if ! command -v tar > /dev/null 2>&1; then
  echo "ERROR: missing tar command, please install"
  exit 1
fi
if ! command -v sudo > /dev/null 2>&1; then
  echo "WARNING: missing sudo command, may be required to elevate permissions so please install or run with relevant permissions"
fi

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
    MINGW64*|Windows)
      echo "windows"
      ;;
    *)
      echo "Unknown OS: $(uname)"
      exit 1
  esac
}

_curl_auth() {
  if [ -n "$GITHUB_TOKEN" ]; then
    curl --header "Authorization: Bearer ${GITHUB_TOKEN}" "$@"
  else
    curl "$@"
  fi
}

_binary_name="calyptia"
if [ "$(_detect_os)" = "windows" ]; then
  _binary_name="calyptia.exe"
  install_dir=$PWD
fi

_download_binary() {
  _download_arch="$(_detect_arch)"
  _download_os="$(_detect_os)"
  # shellcheck disable=SC2154
  _download_version="$CLI_VERSION"
  _download_artefact_prefix="$CLI_ARTEFACT_PREFIX"

  # releases should be prefixed with `v`
  case "$_download_version" in
    "latest") ;;
    "") ;;
    "v"*) ;;
    *)
      _download_version="v$CLI_VERSION"
  esac

  if [ -z "$_download_version" ] || [ "$_download_version" = "latest" ]; then
    _download_version=$(_curl_auth -sSfL "https://api.github.com/repos/${_download_repo}/releases/latest" 2> /dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$_download_version" ]; then
      echo "Unable to retrieve latest CLI version"
      exit 1
    fi
  fi

  _download_trailedVersion="$(echo "$_download_version" | tr -d v)"
  _download_url_prefix="https://github.com/${_download_repo}/releases/download/${_download_version}/${_download_artefact_prefix}_"
  rm -f "$_download_output_dir"/cli.tar.gz

  # macOS does a universal binary in more recent builds so try that as well as the per-arch option
  _url=${_download_url_prefix}${_download_trailedVersion}_${_download_os}_${_download_arch}.tar.gz
  if [ "$_download_os" = "darwin" ]; then
    if ! _curl_auth --output /dev/null --silent --head --fail "$_url"; then
      _url="${_download_url_prefix}${_download_trailedVersion}_${_download_os}_all.tar.gz"
    fi
  fi

  # If we do not have it yet then use the arch version
  echo "Downloading from URL:  $_url"
  _curl_auth --progress-bar --output "$_download_output_dir"/cli.tar.gz -SLf "$_url"
  tar -C "$_download_output_dir" -xzf cli.tar.gz "$_binary_name"
  rm -f "$_download_output_dir"/cli.tar.gz
}

_download_binary

if [ -w "${install_dir}" ]; then
  mv "${_download_output_dir}/$_binary_name" "${install_dir}/$_binary_name"
else
  echo "Sudo rights are needed to move the binary to ${install_dir}, please type your password when asked"
  sudo mv "${_download_output_dir}/$_binary_name" "${install_dir}/$_binary_name"
fi
echo "Calyptia CLI installed to ${install_dir}/$_binary_name"

if [ -n "$CLOUD_API_URL" ]; then
  if [ "$CLOUD_API_URL" != "$("${install_dir}/$_binary_name" current_url)" ]; then
    "${install_dir}/$_binary_name" config unset_token
    echo "Unset any previous token as a precaution"
  fi
  "${install_dir}/$_binary_name" config set_url "$CLOUD_API_URL"
  echo "Using URL: $CLOUD_API_URL"
fi
