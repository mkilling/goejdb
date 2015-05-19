package goejdb

// #cgo LDFLAGS: -lejdb
// #include <ejdb/ejdb.h>
import "C"

import "unsafe"

// Query search mode flags in ejdbqryexecute()
const (
    // Query only count(*)
    jbqrycount   = C.JBQRYCOUNT
    // Fetch first record only
    jbqryfindone = C.JBQRYFINDONE
)

// An EJDB query
type EjQuery struct {
    ptr  *[0]byte
    ejdb *Ejdb
}

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
func (ejdb *Ejdb) CreateQuery(query string, queries ...string) (*EjQuery, *EjdbError) {
    query_bson := bson_from_json(query)
    defer C.bson_destroy(query_bson)

    ptr := C.ejdbcreatequery((*C.struct_EJDB)(unsafe.Pointer(ejdb.ptr)), query_bson, nil, 0, nil)
    if ptr == nil {
        return nil, ejdb.check_error()
    }

    q := &EjQuery{(*[0]byte)(unsafe.Pointer(ptr)), ejdb}

    for _, orquery := range queries {
        err := q.AddOr(orquery)
        if err != nil {
            q.Del()
            return nil, ejdb.check_error()
        }
    }

    return q, nil
}

// Add OR restriction to query object. Expects orquery to be a JSON string.
func (q *EjQuery) AddOr(orquery string) (*EjdbError) {
    bson := bson_from_json(orquery)
    defer C.bson_destroy(bson)
    ret := C.ejdbqueryaddor((*C.struct_EJDB)(unsafe.Pointer(q.ejdb.ptr)), (*C.struct_EJQ)(unsafe.Pointer(q.ptr)), unsafe.Pointer(bson.data))
    if ret == nil {
        return q.ejdb.check_error()
    }
    return nil
}

//  Set query hints. `hints` is a JSON string
//      - $max Maximum number in the result set
//      - $skip Number of skipped results in the result set
//      - $orderby Sorting order of query fields.
//      - $fields Set subset of fetched fields
//          If a field presented in $orderby clause it will be forced to include in resulting records.
//          Example:
//          hints:    {
//                      "$orderby" : { //ORDER BY field1 ASC, field2 DESC
//                          "field1" : 1,
//                          "field2" : -1
//                      },
//                      "$fields" : { //SELECT ONLY {_id, field1, field2}
//                          "field1" : 1,
//                          "field2" : 1
//                      }
//                    }
func (q *EjQuery) SetHints(hints string) *EjdbError {
    bsdata := bson_from_json(hints).data
    ret := C.ejdbqueryhints((*C.struct_EJDB)(unsafe.Pointer(q.ejdb.ptr)), (*C.struct_EJQ)(unsafe.Pointer(q.ptr)), unsafe.Pointer(bsdata))
    if ret == nil {
        return q.ejdb.check_error()
    }
    return nil
}

// Execute the query and return all results as a slice of BSON objects
func (q *EjQuery) Execute(coll *EjColl) ([][]byte, *EjdbError) {
    // execute query
    var count C.uint32_t
    res := C.ejdbqryexecute((*C.struct_EJCOLL)(unsafe.Pointer(coll.ptr)), (*C.struct_EJQ)(unsafe.Pointer(q.ptr)), &count, 0, nil)
    defer C.ejdbqresultdispose(res)
    err := coll.ejdb.check_error()

    // return results
    ret := make([][]byte, 0)
    for i := 0; i < int(count); i++ {
        var size C.int
        bson_blob := C.ejdbqresultbsondata(res, C.int(i), &size)
        el := make([]byte, int(size))
        copy(el, (*[maxslice]byte)(bson_blob)[:int(size)])
        ret = append(ret, el)
    }

    return ret, err
}

// Execute the query and return only the first result as a BSON object
func (q *EjQuery) ExecuteOne(coll *EjColl) (*[]byte, *EjdbError) {
    // execute query
    var count C.uint32_t
    res := C.ejdbqryexecute((*C.struct_EJCOLL)(unsafe.Pointer(coll.ptr)), (*C.struct_EJQ)(unsafe.Pointer(q.ptr)), &count, jbqryfindone, nil)
    defer C.ejdbqresultdispose(res)
    err := coll.ejdb.check_error()

    // return results
    if count == 0 {
        return nil, err
    }

    var size C.int
    bson_blob := C.ejdbqresultbsondata(res, 0, &size)
    ret := make([]byte, int(size))
    copy(ret, ((*[maxslice]byte)(bson_blob))[:int(size)])
    return &ret, err
}

// Execute the query and only return the number of results it returned, not the results themselves
func (q *EjQuery) Count(coll *EjColl) (int, *EjdbError) {
    var count C.uint32_t
    C.ejdbqryexecute((*C.struct_EJCOLL)(unsafe.Pointer(coll.ptr)), (*C.struct_EJQ)(unsafe.Pointer(q.ptr)), &count, jbqrycount, nil)
    err := coll.ejdb.check_error()
    return int(count), err
}

// Delete the query. This must be called in order to not leak memory.
func (q *EjQuery) Del() {
    C.ejdbquerydel((*C.struct_EJQ)(unsafe.Pointer(q.ptr)))
}
