package secretmessage_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSecretmessage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Secretmessage Suite")
}
