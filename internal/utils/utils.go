package utils

import "os"

func GetOperatorNamespace() string {
	namespace := os.Getenv("POD_NAMESPACE")
	return namespace
}
