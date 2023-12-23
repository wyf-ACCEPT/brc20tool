package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

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
	fmt.Println("============ brc20 tool ============")

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

	forEstimate := true
	brc20op := "mint"
	brc20tick := "omni"
	brc20amt := "100"
	brc20repeat := "2"
	feePerBytes := "2"

	txid, txids, fee, err := constructBRC20(forEstimate, brc20op, brc20tick, brc20amt, brc20repeat, feePerBytes)

	if err != nil {
		fmt.Println("Error in running:", err)
		return
	}

	fmt.Println("\n[INFO] transactions (if not simulating):")
	fmt.Println("prepare utxo txid:", txid)
	fmt.Println("inscription txids:", txids)
	fmt.Println("fee:", fee)
}

func constructBRC20(forEstimate bool, brc20op string, brc20tick string, brc20amt string, brc20repeat string, feePerBytes string) (txid string, txids []string, fee int64, err error) {
	// ------------------- Log the basic information -------------------
	fmt.Println("\n[INFO] constructing tx...")
	fmt.Println("is simulate:", forEstimate)

	// ------------------- Define network parameters -------------------
	netParams := &chaincfg.SigNetParams
	btcApiClient := mempool.NewClient(netParams)
	wifKey, err := btcutil.DecodeWIF(privateKey)
	if err != nil {
		return
	}
	fmt.Println("net:", netParams.Name)

	// ------------------- Load the address -------------------
	utxoTaprootAddress, err := btcutil.NewAddressTaproot(schnorr.SerializePubKey(txscript.ComputeTaprootKeyNoScript(wifKey.PrivKey.PubKey())), netParams)
	if err != nil {
		return
	}
	fmt.Println("utxo address:", utxoTaprootAddress.EncodeAddress())

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
	commitTxOutPointList := make([]*wire.OutPoint, 0)
	commitTxPrivateKeyList := make([]*btcec.PrivateKey, 0)
	utxoValues := make([]string, len(unspentList))
	for i := range unspentList {
		utxoValues[i] = fmt.Sprintf("%d", unspentList[i].Output.Value)
		if unspentList[i].Output.Value < 10000 {
			utxoValues[i] = fmt.Sprintf("%s (unusable)", utxoValues[i])
			continue
		}
		commitTxOutPointList = append(commitTxOutPointList, unspentList[i].Outpoint)
		commitTxPrivateKeyList = append(commitTxPrivateKeyList, wifKey.PrivKey)
		vinAmount += int(unspentList[i].Output.Value)
	}
	fmt.Println("utxo count:", len(unspentList))
	fmt.Println("utxo value list: [", strings.Join(utxoValues, ", "), "]")

	// --------------- Construct the inscription data ---------------
	dataList := make([]ord.InscriptionData, 0)
	ordinalText := fmt.Sprintf(`{"p":"brc-20","op":"%s","tick":"%s","amt":"%s"}`, brc20op, brc20tick, brc20amt)
	mint := ord.InscriptionData{
		ContentType: "text/plain;charset=utf-8",
		Body:        []byte(ordinalText),
		Destination: utxoTaprootAddress.EncodeAddress(),
	}
	count, err := strconv.Atoi(brc20repeat)
	if err != nil {
		return
	}
	for i := 0; i < count; i++ {
		dataList = append(dataList, mint)
	}
	fmt.Println("inscription text:", ordinalText)
	fmt.Println("repeat mint times: ", len(dataList))

	// --------------- Construct the inscription request ---------------
	txFee, err := strconv.Atoi(feePerBytes)
	if err != nil {
		return
	}
	request := ord.InscriptionRequest{
		CommitTxOutPointList:   commitTxOutPointList,
		CommitTxPrivateKeyList: commitTxPrivateKeyList,
		CommitFeeRate:          int64(txFee),
		FeeRate:                int64(txFee),
		DataList:               dataList,
		SingleRevealTxOnly:     false,
	}

	// ----------------- Prepare the inscription tool -----------------
	fmt.Println("current balance:", float32(vinAmount)/1e8)
	fmt.Print("total spent fee: ")
	tool, err := ord.NewInscriptionToolWithBtcApiClient(netParams, btcApiClient, &request)
	if err != nil {
		fmt.Println("new tool err: ", err)
		return
	}
	fee = tool.CalculateFee()
	fmt.Println("balance after inscription:", (float32(vinAmount)-float32(fee))/1e8)
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
