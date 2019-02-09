package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/skratchdot/open-golang/open"

	"github.com/HyperspaceApp/Hyperspace/crypto"
	"github.com/HyperspaceApp/Hyperspace/modules"
	"github.com/HyperspaceApp/Hyperspace/types"
	"github.com/HyperspaceApp/fastrand"
)

const outputTmpl = `
<html>
	<head>
		<title> Hyperspace Cold Storage Wallet </title>
	</head>
	<style>
		body {
			font-family: "Gotham A", "Gotham B", Helvetica, Arial, sans-serif;
			margin-left: auto;
			margin-right: auto;
			max-width: 900px;
			text-align: left;
		}
		.info {
			margin-top: 75px;
		}
	</style>
	<body>
		<h2 align="center">Hyperspace Cold Storage Wallet</h3>
		<section class="warning">
			<p> Please write down your seed. Take care not to expose your seed to any potentially insecure device, such as a traditional computer printer. Anyone can use the Seed to recover any Space Cash sent to any of the addresses, without an online or synced wallet. Make sure to keep the seed safe, and secret.</p>
		</section>
		<section class="seed">
			<h4>Seeds</h4>
			{{ range .Seeds }}
				<p><font size="+1">{{.}}</font></p>
			{{ end }}
		</section>
		<section class="addresses">
			<h4>Addresses</h4>
			<ol>
			<font size="+2">
			<code>
			{{ range .Addresses }}
				<li>{{.}}</li>
			{{ end }}
			</code>
			</font>
		</section>
	</body>
	<script>
		window.addEventListener("keydown", function(e) {
			// disable ctrl-p to prevent bad decisions
			if (e.ctrlKey && e.keyCode == 80) {
				e.preventDefault();
				alert("please write down your seed.");
			}
		})
	</script>
</html>
`

const nAddresses = 5
const wordsPerSeed = 29

// getAddress returns an address generated from a seed at the index specified
// by `index`.
func getAddress(seeds []modules.Seed, index, height, n, m uint64) (types.UnlockHash, []types.SiaPublicKey) {
	var pks []types.SiaPublicKey
	for i := 0; i < int(m); i ++ {
		if len(seeds) == 1 {
			_, pk := crypto.GenerateKeyPairDeterministic(crypto.HashAll(seeds[0], index))
			pks = append(pks, types.Ed25519PublicKey(pk))
		} else {
			_, pk := crypto.GenerateKeyPairDeterministic(crypto.HashAll(seeds[i], index))
			pks = append(pks, types.Ed25519PublicKey(pk))
		}
	}
	var uc types.UnlockConditions
	if height != 0 {
		uc = types.UnlockConditions{
			PublicKeys:         pks,
			SignaturesRequired: n,
			Timelock:	    types.BlockHeight(height),
		}
	} else {
		uc = types.UnlockConditions{
			PublicKeys:         pks,
			SignaturesRequired: n,
		}
	}
	return uc.UnlockHash(), pks
}

func main() {
	timelock := flag.Int("timelock", 0, "timelock block height for the addresses")
	n := flag.Int("n", 1, "signatures required")
	m := flag.Int("m", 1, "keys for each address")
	uniqueSeedsPtr := flag.Bool("unique-seeds", false, "use a different seed for each signature")
	printPtr := flag.Bool("print", false, "print pubkeys to cli")
	flag.Parse()

	if *n > *m {
		log.Fatal("Cannot create an address that requires more signatures than there are keys associated with it")
		return
	}

	var seeds []modules.Seed

	// get a seed
	var seedStrs []string
	var words []string
	seedWords := wordsPerSeed
	if *uniqueSeedsPtr {
		seedWords *= *m
	}
	words = flag.Args()
	if len(words) > 0 {
		if len(words) != seedWords {
			log.Fatalf("%v seed words required", seedWords)
		}
		for i := 0; i < *m; i++ {
			curWords := words[i*wordsPerSeed:(i+1)*wordsPerSeed]
			seedStr := strings.Join(curWords[:], " ")
			seed, seedErr := modules.StringToSeed(seedStr, "english")
			if seedErr != nil {
				log.Fatal(seedErr)
			}
			seeds = append(seeds, seed)
			seedStrs = append(seedStrs, seedStr)
		}
	} else {
		for i := 0; i < *m; i++ {
			var seed modules.Seed
			// zero arguments: generate a seed
			fastrand.Read(seed[:])
			seedStr, seedErr := modules.SeedToString(seed, "english")
			if seedErr != nil {
				log.Fatal(seedErr)
			}
			seeds = append(seeds, seed)
			seedStrs = append(seedStrs, seedStr)
		}
	}

	// generate a few addresses from that seed
	var addresses []types.UnlockHash
	if *printPtr {
		for i := 0; i < *m; i++ {
			fmt.Println()
			fmt.Println("Seed:")
			fmt.Println()
			fmt.Println(seedStrs[i])
			fmt.Println()
		}
	}
	for i := uint64(0); i < nAddresses; i++ {
		addr, pubkeys := getAddress(seeds, i, uint64(*timelock), uint64(*n), uint64(*m))
		var keystrs []string
		for _, pubkey := range(pubkeys) {
			keystrs = append(keystrs, pubkey.String())
		}
		addresses = append(addresses, addr)
		if *printPtr {
			fmt.Printf("address %d: %s\n", i, addr)
			fmt.Printf("pubkey %d: %s\n", i, strings.Join(keystrs, ","))
		}
	}

	templateData := struct {
		Seeds     []string
		Addresses []types.UnlockHash
	}{
		Seeds:     seedStrs,
		Addresses: addresses,
	}
	t, err := template.New("output").Parse(outputTmpl)
	if err != nil {
		log.Fatal(err)
	}
	l, err := net.Listen("tcp", "localhost:8087")
	if err != nil {
		log.Fatal(err)
	}

	done := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Execute(w, templateData)
		l.Close()
		close(done)
	})
	go http.Serve(l, handler)

	err = open.Run("http://localhost:8087")
	if err != nil {
		// fallback to console, clean up the server and exit
		l.Close()
		for _, seedStr := range seedStrs {
			fmt.Println("Seed:", seedStr)
		}
		fmt.Println("Addresses:")
		for _, address := range addresses {
			fmt.Println(address)
		}
		os.Exit(0)
	}
	<-done
}
