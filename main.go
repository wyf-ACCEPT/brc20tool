package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/minchenzz/brc20tool/internal/ord"
	"github.com/minchenzz/brc20tool/pkg/btcapi/mempool"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/joho/godotenv"
)

var privateKey string

func main() {
	fmt.Println("======================= brc20 tool =======================")

	// ------------------- Load the private key -------------------
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
		return
	}

	privateKey = os.Getenv("PK")
	if privateKey == "" {
		fmt.Println("Environment variable not found")
		return
	}

	// ------------------- Parse the brc20 params -------------------
	forEstimate := flag.Bool("simulate", false, "Whether to simulate the transaction (true/false). Default is false.")
	brc20op := flag.String("op", "", "BRC20 operation (e.g., `mint`, `transfer` or `deploy`).")
	brc20tick := flag.String("tick", "", "BRC20 ticker symbol (e.g., `ordi`).")
	brc20amt := flag.String("amt", "", "BRC20 amount (e.g., 100).")
	brc20repeat := flag.Int("repeat", 1, "Number of times to repeat the operation. Default is 1.")
	feePerBytes := flag.Int("fee", 10, "Transaction fee per byte. Default is 10.")
	showHelp := flag.Bool("help", false, "Show help.")
	flag.Parse()

	if *showHelp {
		flag.Usage()
		return
	}

	shouldReturn := checkParamsForBRC20(brc20op, brc20tick, brc20amt)
	if shouldReturn {
		return
	}

	// ------------------- Construct the transaction -------------------
	fmt.Println("\n-------------- construct the transaction --------------")
	txid, txids, fee, err := constructBRC20(*forEstimate, *brc20op, *brc20tick, *brc20amt, *brc20repeat, *feePerBytes)

	if err != nil {
		fmt.Println("Error in running:", err)
		return
	}

	if !*forEstimate {
		fmt.Println("\n------------------ transactions info ------------------")
		fmt.Println("prepare utxo txid:", txid)
		fmt.Println("inscription txids:", txids)
		fmt.Println("fee spent        :", fee)
		fmt.Println("view on mempool  : https://mempool.space/signet/tx/" + txid)
	}
}

func checkParamsForBRC20(brc20op *string, brc20tick *string, brc20amt *string) bool {
	if *brc20op == "" {
		fmt.Println("Error: arg --op is required. ")
		return true
	} else if *brc20op != "mint" && *brc20op != "transfer" && *brc20op != "deploy" {
		fmt.Println("Error: arg --op must be `mint`, `transfer` or `deploy`. ")
		return true
	} else if *brc20tick == "" {
		fmt.Println("Error: arg --tick is required. ")
		return true
	} else if *brc20amt == "" {
		fmt.Println("Error: arg --amt is required. ")
		return true
	}
	return false
}

func constructBRC20(forEstimate bool, brc20op string, brc20tick string, brc20amt string, brc20repeat int, feePerBytes int) (txid string, txids []string, fee int64, err error) {
	// ------------------- Log the basic information -------------------
	fmt.Println("is simulate:", forEstimate)

	// ------------------- Define network parameters -------------------
	netParams := &chaincfg.SigNetParams
	btcApiClient := mempool.NewClient(netParams)
	wifKey, err := btcutil.DecodeWIF(privateKey)
	if err != nil {
		return
	}
	fmt.Println("bitcoin net:", netParams.Name)

	// ------------------- Load the address -------------------
	utxoTaprootAddress, err := btcutil.NewAddressTaproot(schnorr.SerializePubKey(txscript.ComputeTaprootKeyNoScript(wifKey.PrivKey.PubKey())), netParams)
	if err != nil {
		return
	}
	fmt.Println("\nyour address:", utxoTaprootAddress.EncodeAddress())

	// --------------------- Collect UTXOs ---------------------
	unspentList, err := btcApiClient.ListUnspent(utxoTaprootAddress)
	if err != nil {
		return
	}
	if len(unspentList) == 0 {
		err = fmt.Errorf("no utxo for %s", utxoTaprootAddress)
		return
	}
	vinAmount := 0
	available := 0
	commitTxOutPointList := make([]*wire.OutPoint, 0)
	commitTxPrivateKeyList := make([]*btcec.PrivateKey, 0)
	utxoValues := make([]string, len(unspentList))
	for i := range unspentList {
		utxoValues[i] = fmt.Sprintf("%d", unspentList[i].Output.Value)
		if unspentList[i].Output.Value < 10000 {
			utxoValues[i] = fmt.Sprintf("%s (unusable)", utxoValues[i])
			continue
		}
		available++
		commitTxOutPointList = append(commitTxOutPointList, unspentList[i].Outpoint)
		commitTxPrivateKeyList = append(commitTxPrivateKeyList, wifKey.PrivKey)
		vinAmount += int(unspentList[i].Output.Value)
	}
	fmt.Println("your utxo count:", len(unspentList), "(", available, "available)")
	// fmt.Println("your utxo value list: [", strings.Join(utxoValues, ", "), "]")

	// --------------- Construct the inscription data ---------------
	dataList := make([]ord.InscriptionData, 0)
	ordinalText := fmt.Sprintf(`{"p":"brc-20","op":"%s","tick":"%s","amt":"%s"}`, brc20op, brc20tick, brc20amt)
	mint := ord.InscriptionData{
		ContentType: "text/plain;charset=utf-8",
		Body:        []byte(ordinalText),
		Destination: utxoTaprootAddress.EncodeAddress(),
	}
	for i := 0; i < brc20repeat; i++ {
		dataList = append(dataList, mint)
	}
	fmt.Println("\nBRC20 inscription text :", ordinalText)
	fmt.Println("BRC20 repeat mint times: ", len(dataList))

	// --------------- Construct the inscription request ---------------
	if err != nil {
		return
	}
	request := ord.InscriptionRequest{
		CommitTxOutPointList:   commitTxOutPointList,
		CommitTxPrivateKeyList: commitTxPrivateKeyList,
		CommitFeeRate:          int64(feePerBytes),
		FeeRate:                int64(feePerBytes),
		DataList:               dataList,
		SingleRevealTxOnly:     false,
	}

	// ----------------- Prepare the inscription tool -----------------
	fmt.Println("\nyour current balance:", float32(vinAmount)/1e8)
	fmt.Print("your total spent fee: ")
	tool, err := ord.NewInscriptionToolWithBtcApiClient(netParams, btcApiClient, &request)
	if err != nil {
		fmt.Println("new tool err: ", err)
		return
	}
	fee = tool.CalculateFee()
	fmt.Println("your balance after inscription:", (float32(vinAmount)-float32(fee))/1e8)
	if forEstimate {
		return
	}

	// ------------------- Submit the transaction -------------------
	commitTxHash, revealTxHashList, _, _, err := tool.Inscribe()
	if err != nil {
		err = fmt.Errorf("send tx errr, %v", err)
		return
	}
	txid = commitTxHash.String()
	for i := range revealTxHashList {
		txids = append(txids, revealTxHashList[i].String())
	}
	return
}
