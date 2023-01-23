# Get the version.
version=`git describe --tags --long`
# Write out the package.
cat << EOF > version.go
package coalago

//go:generate bash ./version.sh
var Version = "$version"
EOF

2022-11-02 03:48:00 > IpWan  { "current-id": "GigabitEthernet1", "description": "At-Hom", "id": "GigabitEthernet1", "order": "0", "priority": "65502", "state": "up" }

2022-11-02 03:48:00 > IpWan  { "current-id": "Wireguard1", "description": "At-Hom", "id": "GigabitEthernet1", "order": "0", "priority": "65502", "state": "down" }

2022-11-02 03:48:00 > IpWan  { "current-id": "Wireguard1", "description": "At-Hom", "id": "GigabitEthernet1", "order": "0", "priority": "65502", "state": "down" }