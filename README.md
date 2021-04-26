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

`GET /scoreboard?address=<string>`

`address` query parameter is optional.
If specified, `me` field is returned together in response.

#### Response

```
{
  "blockHeight" <int>,
  "me": { // optional
    "ranking": <int>,
    "username": <string>,
    "address": <string>,
    "totalScore": <float>,
    "tradingScore": <float>,
    "actionScore": <float>
  }
  "accounts": [
    {
      "ranking": <int>
      "username": <string>,
      "address": <string>,
      "totalScore": <float>,
      "tradingScore": <float>,
      "actionScore": <float>
    }
  ],
  "updatedAt": <string>
}
```

#### Errors

- `404 "account not found"`: Specified account address does not exist in score board.
- `500 "no score board data found"`: There is no server cache of score board.

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

#### Errors

- `500 "no price table data found"`: There is no server cache of price table.
