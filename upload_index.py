#!/bin/env python3

import os,glob,subprocess,json,os.path,urllib.request


def run_cmd(cmd):
    print(f"Running: {cmd}")
    return subprocess.run(cmd.split(), stdout=subprocess.PIPE).stdout.decode('utf-8')

json_target = "https://dstower.home.dolbyn.com/tf/providers/v1/davidjspooner/kubernetes/"
binary_target = "https://dstower.home.dolbyn.com/binary/davidjspooner/terraform-provider-kubernetes/"
gpg_fingerprint = "370CC41578FC61A73584F324A03728FC9B4B6B85"
pgp_public_key = run_cmd(f"gpg --armor --export {gpg_fingerprint}")

version = run_cmd("git describe --exact-match --tags")
print(f"Version: {version}")
version = version.strip("v")

for zip_file in glob.glob("dist/*.zip"):
    sha256 = run_cmd(f"sha256sum {zip_file}").split()[0]
    common = os.path.basename(zip_file)[:-4]
    parts = common.split("_")
    common = "_".join(parts[:-3])
    Version = parts[-3]
    OS = parts[-2]
    Arch = parts[-1]
    body = json.dumps({
        "protocols": [
            "4.0",
            "5.0"
        ],
        "os": f"{OS}",
        "arch": f"{Arch}",
        "filename": f"{common}_{Version}_{OS}_{Arch}.zip",
        "download_url": f"{binary_target}{Version}/{common}_{Version}_{OS}_{Arch}.zip",
        "shasums_url": f"{binary_target}{Version}/{common}_{Version}_SHA256SUMS",
        "shasums_signature_url": f"{binary_target}{Version}/{common}_{Version}_SHA256SUMS.sig",
        "shasum": f"{sha256}",
        "signing_keys": {
            "gpg_public_keys": [
                {
                    "key_id": gpg_fingerprint,
                    "ascii_armor": pgp_public_key,
                    "trust_signature": "",
                    "source": "DavidSpooner <davidjspooner@gmail.com>",
                    "source_url": "https://www.hashicorp.com/security.html"
                }
            ]
        }
    },indent=4)
    with open(f"dist/{common}_{Version}_{OS}_{Arch}.json", "w") as f:
        f.write(body)
    url=f"{json_target}{Version}/download/{OS}/{Arch}"
    print(f"Uploading dist/{common}_{Version}_{OS}_{Arch}.json to {url}")
    req=urllib.request.Request(url,method="PUT", data=body.encode('utf-8'), headers={'Content-Type': 'application/json'})
    with urllib.request.urlopen(req) as f:
        print(f.read().decode('utf-8'))




