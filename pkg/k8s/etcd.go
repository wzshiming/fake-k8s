package k8s

var etcdVersions = map[int]string{
	8:  "3.0.17",
	9:  "3.1.12",
	10: "3.1.12",
	11: "3.2.18",
	12: "3.2.24",
	13: "3.2.24",
	14: "3.3.10",
	15: "3.3.10",
	16: "3.3.17-0",
	17: "3.4.3-0",
	18: "3.4.3-0",
	19: "3.4.13-0",
	20: "3.4.13-0",
	21: "3.4.13-0",
	22: "3.5.1-0",
	23: "3.5.1-0",
	24: "3.5.1-0",
}

func GetEtcdVersion(version int) string {
	if version < 8 {
		version = 8
	}
	if version > 24 {
		version = 24
	}
	return etcdVersions[version]
}
