EJDB Go binding [![Build Status](https://travis-ci.org/mkilling/goejdb.png?branch=master)](https://travis-ci.org/mkilling/goejdb)
==================================

One snippet intro
-----------------------------------

```go
package ejdbtutorial

import (
    "fmt"
    "github.com/mkilling/goejdb"
    "labix.org/v2/mgo/bson"
    "os"
)

func main() {
    // Create a new database file and open it
    jb, err := goejdb.Open("addressbook", JBOWRITER | JBOCREAT | JBOTRUNC)
    if err != nil {
        os.Exit(1)
    }
    // Get or create collection 'contacts'
    coll, _ := jb.CreateColl("contacts", nil)

    // Insert one record:
    // JSON: {'name' : 'Bruce', 'phone' : '333-222-333', 'age' : 58}
    rec := map[string]interface{} {"name" : "Bruce", "phone" : "333-222-333", "age" : 58}
    bsrec, _ := bson.Marshal(rec)
    coll.SaveBson(bsrec)
    fmt.Printf("\nSaved Bruce")

    // Now execute query
    res, _ := coll.Find(`{"name" : {"$begin" : "Bru"}}`) // Name starts with 'Bru' string
    fmt.Printf("\n\nRecords found: %d\n", len(res))

    // Now print the result set records
    for _, bs := range res {
        var m map[string]interface{}
        bson.Unmarshal(bs, &m)
        fmt.Println(m)
    }

    // Close database
    jb.Close()
}
```

You can save this code in `ejdbtutorial.go` and build:


```sh
go build ejdbtutorial.go
./ejdbtutorial
```

Installation
-------------------------------

### Prerequisites
**System libraries:**

* Google Go
* installed ejdb (see [https://github.com/Softmotions/ejdb](https://github.com/Softmotions/ejdb) or [Installing on Debian/Ubuntu](https://github.com/Softmotions/ejdb/wiki/Debian-Ubuntu-installation))

### Install

    go get github.com/mkilling/goejdb

Queries
---------------------------------

```go
// Create query object.
// Sucessfully created queries must be destroyed with Query.Del().
//
// EJDB queries inspired by MongoDB (mongodb.org) and follows same philosophy.
//
//  - Supported queries:
//      - Simple matching of String OR Number OR Array value:
//          -   {'fpath' : 'val', ...}
//      - $not Negate operation.
//          -   {'fpath' : {'$not' : val}} //Field not equal to val
//          -   {'fpath' : {'$not' : {'$begin' : prefix}}} //Field not begins with val
//      - $begin String starts with prefix
//          -   {'fpath' : {'$begin' : prefix}}
//      - $gt, $gte (>, >=) and $lt, $lte for number types:
//          -   {'fpath' : {'$gt' : number}, ...}
//      - $bt Between for number types:
//          -   {'fpath' : {'$bt' : [num1, num2]}}
//      - $in String OR Number OR Array val matches to value in specified array:
//          -   {'fpath' : {'$in' : [val1, val2, val3]}}
//      - $nin - Not IN
//      - $strand String tokens OR String array val matches all tokens in specified array:
//          -   {'fpath' : {'$strand' : [val1, val2, val3]}}
//      - $stror String tokens OR String array val matches any token in specified array:
//          -   {'fpath' : {'$stror' : [val1, val2, val3]}}
//      - $exists Field existence matching:
//          -   {'fpath' : {'$exists' : true|false}}
//      - $icase Case insensitive string matching:
//          -    {'fpath' : {'$icase' : 'val1'}} //icase matching
//          Ignore case matching with '$in' operation:
//          -    {'name' : {'$icase' : {'$in' : ['théâtre - театр', 'hello world']}}}
//          For case insensitive matching you can create special index of type: `JBIDXISTR`
//      - $elemMatch The $elemMatch operator matches more than one component within an array element.
//          -    { array: { $elemMatch: { value1 : 1, value2 : { $gt: 1 } } } }
//          Restriction: only one $elemMatch allowed in context of one array field.
//
//  - Queries can be used to update records:
//
//      $set Field set operation.
//          - {.., '$set' : {'field1' : val1, 'fieldN' : valN}}
//      $upsert Atomic upsert. If matching records are found it will be '$set' operation,
//              otherwise new record will be inserted
//              with fields specified by argment object.
//          - {.., '$upsert' : {'field1' : val1, 'fieldN' : valN}}
//      $inc Increment operation. Only number types are supported.
//          - {.., '$inc' : {'field1' : number, ...,  'field1' : number}
//      $dropall In-place record removal operation.
//          - {.., '$dropall' : true}
//      $addToSet Atomically adds value to the array only if its not in the array already.
//                If containing array is missing it will be created.
//          - {.., '$addToSet' : {'fpath' : val1, 'fpathN' : valN, ...}}
//      $addToSetAll Batch version if $addToSet
//          - {.., '$addToSetAll' : {'fpath' : [array of values to add], ...}}
//      $pull  Atomically removes all occurrences of value from field, if field is an array.
//          - {.., '$pull' : {'fpath' : val1, 'fpathN' : valN, ...}}
//      $pullAll Batch version of $pull
//          - {.., '$pullAll' : {'fpath' : [array of values to remove], ...}}
//
// - Collection joins supported in the following form:
//
//      {..., $do : {fpath : {$join : 'collectionname'}} }
//      Where 'fpath' value points to object's OIDs from 'collectionname'. Its value
//      can be OID, string representation of OID or array of this pointers.
//
//  NOTE: Negate operations: $not and $nin not using indexes
//  so they can be slow in comparison to other matching operations.
//
//  NOTE: Only one index can be used in search query operation.
func (ejdb *Ejdb) CreateQuery(query string, queries ...string) (*EjQuery, *EjdbError)
```
