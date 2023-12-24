# BRC20 constructor

This is a **BRC20 transaction constructor**, forked from [eastonqiu/brc20tool](https://github.com/eastonqiu/brc20tool/tree/main).

## Usage

### 1. Prepare the environment

You should have [Go](https://golang.org/) installed on your computer. If not, follow this [guidance](https://go.dev/doc/install) to install.

### 2. Prepare a wallet

You can use [Unisat](https://unisat.io/) to create a wallet, and switch to Signet network for test. Use [this faucet](https://signet.bc-2.jp/) to get some Signet BTC. Remeber to create a `.env` file and put your wallet's private key in it.

You should fill in the `.env` file like this:

```bash
PK="1111YourPrivateKeyIsABase58EncodedStringWithLength52"
```

### 3. Run the script

You can deliver your BRC20 params into as command line arguments. For example:

```bash
go run main.go --op mint --tick sats --amt 10000000 --network signet --simulate
```

You can also use the `--help` flag to get more information. The help message is as follows:

```log
Usage of ./main.go:
  -amt string
        BRC20 amount (e.g., 100).
  -fee int
        Transaction fee per byte. Default is 10. (default 10)
  -help
        Show help.
  -network mainnet
        Bitcoin network (e.g., mainnet, `testnet` or `signet`). Default is `signet`. (default "signet")
  -op mint
        BRC20 operation (e.g., mint or `transfer`).
  -repeat int
        Number of times to repeat the operation. Default is 1. (default 1)
  -simulate
        Whether to simulate the transaction (true/false). Default is false.
  -tick ordi
        BRC20 ticker symbol (e.g., ordi).

```