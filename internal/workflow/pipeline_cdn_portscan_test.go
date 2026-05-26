package workflow

import "testing"

func TestIpsForPortScan(t *testing.T) {
	all := []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"}
	nonCDN := []string{"1.1.1.1"}

	t.Run("no classification", func(t *testing.T) {
		got := ipsForPortScan(all, nonCDN, false, true)
		if len(got) != 3 {
			t.Fatalf("got %v, want all IPs", got)
		}
	})

	t.Run("skip cdn hosts", func(t *testing.T) {
		got := ipsForPortScan(all, nonCDN, true, true)
		if len(got) != 1 || got[0] != "1.1.1.1" {
			t.Fatalf("got %v, want non-CDN only", got)
		}
	})

	t.Run("scan cdn when skip disabled", func(t *testing.T) {
		got := ipsForPortScan(all, nonCDN, true, false)
		if len(got) != 3 {
			t.Fatalf("got %v, want all IPs including CDN", got)
		}
	})
}
