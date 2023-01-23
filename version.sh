# Get the version.
version=`git describe --tags --long`
# Write out the package.
cat << EOF > version.go
package coalago