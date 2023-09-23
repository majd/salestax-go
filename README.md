# salestax-go

This repository contains a library for performing international sales tax calculation, such as for countries in the European Union (VAT MOSS), completely offline.
It is based on the [node-sales-tax](https://github.com/valeriansaliou/node-sales-tax) library for NodeJS.

## Usage

```go
package main

import (
	"fmt"
	"os"

	"github.com/majd/salestax-go"
)

func main() {
	ctrl := &salestax.Ctrl{
		OriginCountryCode:  salestax.Ptr("DE"),
		RegionalTaxEnabled: true,
	}

	tax, err := ctrl.GetSalesTax("DE", nil, nil)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	fmt.Printf("%+v\n", tax)
}
```

## License

The code in this library is released under the [MIT license](https://github.com/majd/salestax-go/blob/main/LICENSE).