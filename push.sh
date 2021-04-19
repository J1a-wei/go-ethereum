#!/usr/bin/env bash

set -e -u

cd "$(dirname "$0")"

os=`uname`

project_name=geth
tail=tgz
project_version=1.8.15
user=deployer
password=n8y6XoZ34HQN
url="http://nexus-1a-1.aws-jp1.huobiidc.com:8081"
maven_release="repository/maven-releases"
group="com/huobi"
artifact="geth"
version=`git describe --always`
unix_time=`date +%s`
version_url=${url}/${maven_release}/${group}/${artifact}/${unix_time}
main_name=${project_name}-${unix_time}.${tail}

cd build/bin

if [[ ${os} == "Darwin" ]]; then
    md5 geth | awk '{ print $4 }' > geth.md5;
elif [[ ${os} == "Linux" ]]; then
    md5sum geth | awk '{ print $1 }' > geth.md5;
else
    echo "${os} not support now" > geth.md5;
fi

tar czvf geth.${tail} geth geth.md5

if [[ ${os} == "Darwin" ]]; then
    md5 geth.${tail} | awk '{ print $4 }' > geth.${tail}.md5;
elif [[ ${os} == "Linux" ]]; then
    md5sum  geth.${tail} | awk '{ print $1 }' > geth.${tail}.md5;
else
    echo "${os} not support now" > geth.${tail}.md5;
fi

git log -1 > geth.${tail}.commit

curl -v -u ${user}:${password} --upload-file geth.${tail}.commit ${version_url}/${main_name}.commit
curl -v -u ${user}:${password} --upload-file geth.${tail}.md5 ${version_url}/${main_name}.md5
curl -v -u ${user}:${password} --upload-file geth.${tail} ${version_url}/${main_name}

echo ""
echo -e "\x1b[36m${version_url}/${main_name}\x1b[0m"
echo ""

