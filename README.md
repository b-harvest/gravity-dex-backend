# gravity-dex-backend

## Prerequisites

Please install the latest version of:
- Go (1.x)
- MongoDB (4.x)
- Redis (6.x)

## Build & Install

```
$ go install ./...
```

This will build and install `gdex` binary in `$GOPATH/bin`.

## Usage

### Configuration

Transformer and Server requires a configuration file, `config.yml`, in current working directory.
All available configurations can be found in [here](./config/config.go)

### Transformer

Transformer keeps reading `transformer.block_data_dir` and synchronizes chain's state with the database.
Run it in background:
```
$ gdex transformer
```

### Server

Server is the API server.
It generates score board and price table then caches those in background.
Run it with:
```
$ gdex server
```

## API Endpoints

### Score Board

#### Request

`GET /scoreboard`

#### Response

```
{
  "accounts": [
    {
      "username": <string>,
      "address": <string>,
      "totalScore": <float>,
      "tradingScore": <float>,
      "actionScore": <float>
    }
  ]
}
```

### Price Table

#### Request

`GET /pricetable`

#### Response

```
{
  "pools": [
    {
      "id": <uint64>,
      "reserveCoins": [
        {
          "denom": <string>, // lowercased
          "amount": <int>,
          "globalPrice": <float>
        },
        {
          "denom": <string>,
          "amount": <int>,
          "globalPrice": <float>
        }
      ],
    }
  ]
}
```
