package coalago_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestCoalago(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Coalago Suite")
}
