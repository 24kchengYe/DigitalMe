package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/chenhg5/cc-connect/licensing"
)

func runLicenseCommand(args []string) {
	if len(args) == 0 {
		printLicenseUsage()
		return
	}

	switch args[0] {
	case "gen", "generate":
		runLicenseGen(args[1:])
	case "verify":
		runLicenseVerify(args[1:])
	case "info":
		runLicenseInfo()
	default:
		fmt.Fprintf(os.Stderr, "Unknown license subcommand: %s\n", args[0])
		printLicenseUsage()
		os.Exit(1)
	}
}

func printLicenseUsage() {
	fmt.Println(`Usage: digitalme license <subcommand>

Subcommands:
  gen       Generate a license key
  verify    Verify a license key
  info      Show current license status

Examples:
  digitalme license gen --licensee "Acme Corp" --email "admin@acme.com" --tier pro --expires 2027-01-01
  digitalme license verify <key>
  digitalme license info`)
}

func runLicenseGen(args []string) {
	fs := flag.NewFlagSet("license gen", flag.ExitOnError)
	licensee := fs.String("licensee", "", "Licensee name (required)")
	email := fs.String("email", "", "Licensee email (required)")
	tier := fs.String("tier", "pro", "License tier: free or pro")
	expires := fs.String("expires", "", "Expiration date (YYYY-MM-DD), empty = perpetual")
	secret := fs.String("secret", "", "Signing key (hex), or set DIGITALME_SIGN_KEY env var")
	fs.Parse(args)

	if *licensee == "" || *email == "" {
		fmt.Fprintln(os.Stderr, "Error: --licensee and --email are required")
		fs.Usage()
		os.Exit(1)
	}

	// Default features for pro tier
	var features []string
	if *tier == "pro" {
		features = []string{
			licensing.FeatureVoice,
			licensing.FeatureFileSendback,
			licensing.FeatureCron,
			licensing.FeatureWebDashboard,
			licensing.FeatureScreenshot,
			licensing.FeatureTaskNotify,
		}
	}

	payload := licensing.Payload{
		Licensee: *licensee,
		Email:    *email,
		Tier:     licensing.Tier(*tier),
		Expires:  *expires,
		Features: features,
	}

	key, err := licensing.GenerateKey(payload, *secret)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("License Key:")
	fmt.Println(key)
	fmt.Println()
	fmt.Printf("Licensee: %s\n", *licensee)
	fmt.Printf("Email:    %s\n", *email)
	fmt.Printf("Tier:     %s\n", *tier)
	if *expires != "" {
		fmt.Printf("Expires:  %s\n", *expires)
	} else {
		fmt.Println("Expires:  never")
	}
}

func runLicenseVerify(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: digitalme license verify <key>")
		os.Exit(1)
	}

	keyStr := args[0]
	payload, err := licensing.VerifyKeyString(keyStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "INVALID: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("VALID")
	data, _ := json.MarshalIndent(payload, "", "  ")
	fmt.Println(string(data))

	if payload.IsExpired() {
		fmt.Println("\nWARNING: This key has expired.")
	}
}

func runLicenseInfo() {
	fmt.Println(licensing.StatusString())
}
