#!/bin/sh

#set -x

DIR="`dirname \"$0\"`"
ROOTDIR="`( cd \"$DIR/../\" && pwd )`"

cfssl=${ROOTDIR}/.tools/bin/cfssl
cfssljson=${ROOTDIR}/.tools/bin/cfssljson

! [ -e "$cfssl" ] && echo "cfssl not found. run 'make install-deptools'" && exit 1
! [ -e "$cfssljson" ] && echo "cfssl not found. run 'make install-deptools'" && exit 1

_cert_dir=${ROOTDIR}/.certs
rm -rf ${_cert_dir} && mkdir -p ${_cert_dir}

HOSTNAMES=localhost,127.0.0.1

_expiry="720h"
_ca_name="example-app-ca"
_algo="rsa"
_size=2048

_tmpdir=$(mktemp -d)
_ca_config_json=${_tmpdir}/ca_config.json

cat > ${_ca_config_json} <<EOF
{
  "signing": {
    "default": {
      "expiry": "${_expiry}"
    },
    "profiles": {
      "example-app-server": {
        "usages": ["signing", "key encipherment", "server auth"],
        "expiry": "${_expiry}"
      },
      "example-app-client": {
        "usages": ["signing", "key encipherment", "client auth"],
        "expiry": "${_expiry}"
      }
    }
  }
}
EOF

_ca_csr_json=${_tmpdir}/ca_csr.json
cat > ${_ca_csr_json} <<EOF
{
  "CN": "${_ca_name}",
  "key": {
    "algo": "${_algo}",
    "size": ${_size}
  },
  "names": [
    {
      "C": "US",
      "ST": "California",
      "L": "San Francisco",
      "O": "Example",
      "OU": "Example Inc"
    }
  ]
}
EOF

# generate CA
mkdir -p ${_cert_dir}/ca
$cfssl gencert -loglevel=5 -initca ${_ca_csr_json} | $cfssljson -bare ${_cert_dir}/ca/tls

for server_component in reports-server; do
cat > ${_tmpdir}/${server_component}-csr.json <<EOF
{
  "CN": "example-app-${server_component}",
  "key": {
    "algo": "${_algo}",
    "size": ${_size}
  },
  "names": [
    {
      "C": "US",
      "ST": "California",
      "L": "San Francisco",
      "O": "Example",
      "OU": "Example Inc"
    }
  ]
}
EOF

mkdir -p ${_cert_dir}/${server_component}
# generate server
$cfssl gencert \
  -ca=${_cert_dir}/ca/tls.pem \
  -ca-key=${_cert_dir}/ca/tls-key.pem \
  -config=${_ca_config_json} \
  -hostname=${HOSTNAMES} \
  -profile=example-app-server \
  -loglevel=5 \
  ${_tmpdir}/${server_component}-csr.json | $cfssljson -bare ${_cert_dir}/${server_component}/tls

done

# generate example-app client server
for client_component in cli; do
cat > ${_tmpdir}/${client_component}-csr.json <<EOF
{
  "CN": "example-app-${client_component}",
  "key": {
    "algo": "${_algo}",
    "size": ${_size}
  },
  "names": [
    {
      "C": "US",
      "ST": "California",
      "L": "San Francisco",
      "O": "Example",
      "OU": "Example Inc"
    }
  ]
}
EOF

mkdir -p ${_cert_dir}/${client_component}

$cfssl gencert \
  -ca=${_cert_dir}/ca/tls.pem \
  -ca-key=${_cert_dir}/ca/tls-key.pem \
  -config=${_ca_config_json} \
  -profile=example-app-client \
  -loglevel=5 \
  ${_tmpdir}/${client_component}-csr.json | $cfssljson -bare ${_cert_dir}/${client_component}/tls

done

find ${_cert_dir} -name '*.csr' | xargs rm
find ${_cert_dir} -name 'tls-key.pem' -exec bash -c 'mv $0 `dirname $0`/tls.key' '{}' \;
find ${_cert_dir} -name 'tls.pem'  -exec bash -c 'mv $0 `dirname $0`/tls.crt' '{}' \;
