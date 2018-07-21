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

	"github.com/NebulousLabs/Sia/crypto"
	"github.com/NebulousLabs/Sia/modules"
	"github.com/NebulousLabs/Sia/types"
	"github.com/NebulousLabs/fastrand"
)

const outputTmpl = `
<html>
	<head>
		<title> Sia Cold Storage Wallet </title>
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
		<h2 align="center">Sia Cold Storage Wallet</h3>
		<section class="warning">
			<p> Please write down your seed. Take care not to expose your seed to any potentially insecure device, such as a traditional computer printer. Anyone can use the Seed to recover any Siacoin sent to any of the addresses, without an online or synced wallet. Make sure to keep the seed safe, and secret.</p>
		</section>
		<section class="seed">
			<h4>Seed</h4>
			<p><font size="+1">{{.Seed}}</font></p>
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

const nAddresses = 20

// getAddress returns an address generated from a seed at the index specified
// by `index`.
func getAddress(seed modules.Seed, index, height, n, m uint64) types.UnlockHash {
	var pks []types.SiaPublicKey
	for i := 0; i < int(m); i ++ {
		_, pk := crypto.GenerateKeyPairDeterministic(crypto.HashAll(seed, index))
		pks = append(pks, types.Ed25519PublicKey(pk))
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
	return uc.UnlockHash()
}

func main() {
	timelock := flag.Int("timelock", 0, "timelock block height for the addresses")
	n := flag.Int("n", 1, "signatures required")
	m := flag.Int("m", 1, "keys for each address")
	flag.Parse()

	if *n > *m {
		log.Fatal("Cannot create an address that requires more signatures than there are keys associated with it")
		return
	}

	var seed modules.Seed

	// get a seed
	var seedErr error
	var seedStr string
	var words []string
	words = flag.Args()
	if len(words) > 0 {
		if len(words) != 29 {
			log.Fatal("29 seed words required")
		}
		seedStr = strings.Join(words[:], " ")
		seed, seedErr = modules.StringToSeed(seedStr, "english")
	} else {
		// zero arguments: generate a seed
		fastrand.Read(seed[:])
		seedStr, seedErr = modules.SeedToString(seed, "english")
	}
	if seedErr != nil {
		log.Fatal(seedErr)
	}

	// generate a few addresses from that seed
	var addresses []types.UnlockHash
	for i := uint64(0); i < nAddresses; i++ {
		addresses = append(addresses, getAddress(seed, i, uint64(*timelock), uint64(*n), uint64(*m)))
	}

	templateData := struct {
		Seed      string
		Addresses []types.UnlockHash
	}{
		Seed:      seedStr,
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
		fmt.Println("Seed:", seedStr)
		fmt.Println("Addresses:")
		for _, address := range addresses {
			fmt.Println(address)
		}
		os.Exit(0)
	}
	<-done
}
