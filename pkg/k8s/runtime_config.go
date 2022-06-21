package k8s

func GetRuntimeConfig(version int) string {
	if version < 17 {
		return ""
	}
	return "api/legacy=false,api/alpha=false"
}
