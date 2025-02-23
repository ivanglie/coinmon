# Coinmon

A service for retrieving cryptocurrency data from multiple exchanges. Can be integrated with spreadsheets, monitoring tools and trading dashboards.

## Features

- **Simple approach**: REST API compatible with Microsoft Excel, Google Sheets and similar tools
- **Anonymous access**: No sign ups required, no API keys required
- **Multiple sources**: Data from major exchanges such as Binance, Bybit, Bitget
- **Efficient implementation**: Concurrent request processing
- **Configurable output**: Basic price or detailed response format

## Usage

API endpoints:
```
https://coinmon.cc/api/v1/spot/BTCUSDT         # Returns price value
https://coinmon.cc/api/v1/spot/BTCUSDT?details=true  # Returns detailed JSON
```
API basic response:
```
96297.49
```

API detailed response Format:
```json
{
    "pair": "BTCUSDT",
    "price": 96297.49,
    "source": "binance"
}
```

### Spreadsheet Integration

Microsoft Excel:
```
=WEBSERVICE("https://coinmon.cc/api/v1/spot/BTCUSDT")
```

Google Sheets:
```
=IMPORTDATA("https://coinmon.cc/api/v1/spot/BTCUSDT")
```

Google Apps Script implementation:
```javascript
function getCryptoPrice() {
  var response = UrlFetchApp.fetch("https://coinmon.cc/api/v1/spot/BTCUSDT");
  return parseFloat(response.getContentText());
}
```

### Supported Trading Pairs

Compatible with standard exchange trading pairs:
- BTCUSDT
- ETHUSDT
- SOLUSDT
- etc.

## Technical Details

The Coinmon service is also suitable as a self-hosted solution.

### Installation

```bash
# Clone the repository
git clone https://github.com/ivanglie/coinmon.git

# Change to project directory
cd coinmon

# Install dependencies
go mod tidy
```

### Running the Service

Local environment:
```bash
make run
```

Docker environment:
```bash
make docker-dev   # development
make docker-prod  # production
```

## License

[MIT License](/LICENSE.md)