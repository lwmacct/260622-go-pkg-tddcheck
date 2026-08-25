package filelayout

import "github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"

type fileName = rulekit.FileIdentity

func parseFileName(base string, mode string) (fileName, bool) {
	return rulekit.ParseFileIdentity("", base, mode)
}
