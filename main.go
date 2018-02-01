package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	b "github.com/stellar/go/build"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/network"
)

type config struct {
	client  *horizon.Client
	network b.Network
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func readConfig(cpath string) *config {

	/*TODO: add custom network support
	&config{
		client: &http.Client{
			URL:  customNetworkURL
			HTTP: http.DefaultClient,
		}

		network: b.Network{customPassphrase}}
	*/

	switch snet := getEnv("STELLAR_NETWORK", "test"); snet {
	case "public":
		return &config{
			client:  horizon.DefaultPublicNetClient,
			network: b.Network{network.PublicNetworkPassphrase}}
	case "test":
		return &config{
			client:  horizon.DefaultTestNetClient,
			network: b.Network{network.TestNetworkPassphrase}}
	default:
		log.Fatalf("Unknown Stellar network: \"%s\". Stellar network is set by the \"STELLAR_NETWORK\" environment variable. Possible values are \"public\", \"test\". An unset \"STELLAR_NETWORK\" is treated as \"test\".", snet)
	}

	return nil
}

// ./mc --gen-keys foo; ./mc --fund $(cat foo.pub)
func main() {
	var fund string
	var keyFpath string
	var txToSubmit string
	var setTrustline string
	var sendPayment string
	var txOptions string
	var accountDetails string
	var buildTransaction string

	flag.StringVar(&fund, "fund", "", "fund a test account. example: --fund address")
	flag.StringVar(&keyFpath, "gen-keys", "", "create a pair of keys (in two files \"file-path\" and \"file-path.pub\"). example: --gen-keys file-path")
	flag.StringVar(&txToSubmit, "submit-tx", "", "submit a base64 encoded transaction. example: --submit-tx txn")
	flag.StringVar(&setTrustline, "change-trust", "", "create, update, or delete a trustline. has a \"limit\" param which is optional, setting it to \"0\" removes the trustline example: --change-trust '{\"source-account\": \"seed\", \"code\": \"XYZ\", \"issuer-address\": \"address\", \"limit\": \"42.0\"}'")
	flag.StringVar(&sendPayment, "send-payment", "", "send payment from one account to another. example: --send-payment '{\"from\": \"seed\", \"to\": \"address\", \"token\": \"BTC\", \"amount\": \"42.0\", \"issuer\": \"address\"}'")
	flag.StringVar(&accountDetails, "account-details", "", "load and return account details. example: --account-details address")
	flag.StringVar(&txOptions, "tx-options", "", "add one or more transaction options. example: --tx-options '{\"home-domain\": \"stellar.org\", \"max-weight\": 1, \"inflation-destination\": \"address\"}'")
	flag.StringVar(&buildTransaction, "new-tx", "", "build and submit a new transaction. \"operations\" and \"signers\" are optional, if there are no \"signers\", the \"source-account\" seed will be used to sign this transaction. example: --new-tx '{\"source-account\": \"address or seed\", {\"operations\": \"trust\": {\"code\": \"XYZ\", \"issuer-address\": \"address\"}}, \"signers\": [\"seed1\", \"seed2\"]}'")

	flag.Parse()

	conf := readConfig("/tmp/todo")

	var txOptionsBuilder b.SetOptionsBuilder
	if txOptions != "" {
		txOptionsBuilder = parseOptions(txOptions)
	}

	switch {
	case fund != "":
		fundTestAccount(conf.client, fund)
	case keyFpath != "":
		createNewKeys(keyFpath)
	case txToSubmit != "":
		submitTransactionB64(conf.client, txToSubmit)
	case accountDetails != "":
		fmt.Println(toJSON(loadAccount(conf.client, accountDetails)))
	case sendPayment != "":
		payment := &tokenPayment{}
		if err := json.Unmarshal([]byte(sendPayment), payment); err != nil {
			log.Fatal(err)
		}
		tx := payment.send(conf, txOptionsBuilder)
		submitTransaction(conf.client, tx, payment.From)
	case setTrustline != "":
		ct := &changeTrust{}
		if err := json.Unmarshal([]byte(setTrustline), ct); err != nil {
			log.Fatal(err)
		}
		tx := ct.set(conf, txOptionsBuilder)
		submitTransaction(conf.client, tx, ct.SourceAccount)
	case buildTransaction != "":
		nt := &newTransaction{}
		if err := json.Unmarshal([]byte(buildTransaction), nt); err != nil {
			log.Fatal(err)
		}
		nt.Operations.SourceAccount = &b.SourceAccount{nt.SourceAccount}
		tx := nt.Operations.buildTransaction(conf, txOptionsBuilder)
		signers := nt.Signers
		if signers == nil {
			signers = []string{nt.SourceAccount}
		}
		submitTransaction(conf.client, tx, signers...)
	default:
		if txOptions != "" {
			fmt.Errorf("\"--tx-options\" can't be used by itself, it is an additional flag that should be used with other flags that build transactions: i.e. \"--send-payment ... --tx-options ...\" or \"--change-trust ... --tx-options ...\"")
		} else {
			flag.PrintDefaults()
		}
	}
}
