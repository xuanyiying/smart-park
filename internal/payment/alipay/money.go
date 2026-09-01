package alipay

import "fmt"

// centsToYuanString renders integer cents (分) as the two-decimal yuan string
// the Alipay gateway exchanges, without routing the value through binary
// floating point. 1234 -> "12.34", 5 -> "0.05".
func centsToYuanString(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return fmt.Sprintf("%s%d.%02d", sign, cents/100, cents%100)
}
