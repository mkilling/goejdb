package goejdb

// #cgo LDFLAGS: -lejdb
// #include <ejdb/ejdb.h>
import "C"

import "unsafe"

// Index modes, index types.
const (
    // Drop index.
    JBIDXDROP    = C.JBIDXDROP
    // Drop index for all types.
    JBIDXDROPALL = C.JBIDXDROPALL
    // Optimize indexes.
    JBIDXOP      = C.JBIDXOP
    // Rebuild index.
    JBIDXREBLD   = C.JBIDXREBLD
    // Number index.
    JBIDXNUM     = C.JBIDXNUM
    // String index.*/
    JBIDXSTR     = C.JBIDXSTR
    // Array token index.
    JBIDXARR     = C.JBIDXARR
    // Case insensitive string index
    JBIDXISTR    = C.JBIDXISTR
)

// An EJDB collection
type EjColl struct {
    ptr  *[0]byte
    ejdb *Ejdb
}

func (coll *EjColl) save_c_bson(c_bson *C.bson) (string, *EjdbError) {
    var c_oid C.bson_oid_t
    C.ejdbsavebson((*C.struct_EJCOLL)(unsafe.Pointer(coll.ptr)), c_bson, &c_oid)
    return bson_oid_to_string(&c_oid), coll.ejdb.check_error()
}

// Persist BSON object in the collection.
// If saved bson does't have _id primary key then `oid` will be set to generated bson _id,
// otherwise `oid` will be set to the current bson's _id field.
//
// NOTE: Field names of passed `bs` object may not contain `$` and `.` characters,
//       error condition will be fired in this case.
//
// BSON object must be a byte slice, for instance one created by the "labix.org/v2/mgo/bson" library
func (coll *EjColl) SaveBson(bsdata []byte) (string, *EjdbError) {
    c_bson := bson_from_byte_slice(bsdata)
    defer C.bson_destroy(c_bson)
    return coll.save_c_bson(c_bson)
}

// Persist JSON string in the collection.
// If saved json does't have _id primary key then `oid` will be set to generated bson _id,
// otherwise `oid` will be set to the current json's _id field.
//
// NOTE: Integer literals in the JSON string will be converted to float64
func (coll *EjColl) SaveJson(j string) (string, *EjdbError) {
    c_bson := bson_from_json(j)
    defer C.bson_destroy(c_bson)
    return coll.save_c_bson(c_bson)
}

// EJDB_EXPORT bool ejdbrmbson(EJCOLL *coll, bson_oid_t *oid);
func (coll *EjColl) RmBson(oid string) bool {
    c_oid := bson_oid_from_string(&oid)
    ret := C.ejdbrmbson((*C.struct_EJCOLL)(unsafe.Pointer(coll.ptr)), c_oid)
    coll.ejdb.check_error()
    return bool(ret)
}

// EJDB_EXPORT bson* ejdbloadbson(EJCOLL *coll, const bson_oid_t *oid);
func (coll *EjColl) LoadBson(oid string) []byte {
    c_oid := bson_oid_from_string(&oid)
    bson := C.ejdbloadbson((*C.struct_EJCOLL)(unsafe.Pointer(coll.ptr)), c_oid)
    defer C.bson_del(bson)
    coll.ejdb.check_error()

    return bson_to_byte_slice(bson)
}

// Execute a query specified by JSON strings query, queries and return the results as a slice of BSON objects
// See the documentation of EjQuery  for a description of the query format.
func (coll *EjColl) Find(query string, queries ...string) ([][]byte, *EjdbError) {
    q, err := coll.ejdb.CreateQuery(query, queries...)
    defer q.Del()
    if err != nil {
        return nil, err
    }
    return q.Execute(coll)
}

// Execute a query specified by JSON strings query, queries and return only the first result as a BSON object
// See the documentation of EjQuery  for a description of the query format.
func (coll *EjColl) FindOne(query string, queries ...string) (*[]byte, *EjdbError) {
    q, err := coll.ejdb.CreateQuery(query, queries...)
    defer q.Del()
    if err != nil {
        return nil, err
    }
    return q.ExecuteOne(coll)
}

// Execute a query specified by JSON strings query, queries and return the number of results, not the results themselves.
// See the documentation of EjQuery  for a description of the query format.
func (coll *EjColl) Count(query string, queries ...string) (int, *EjdbError) {
    q, err := coll.ejdb.CreateQuery(query, queries...)
    if err != nil {
        return 0, err
    }
    defer q.Del()
    return q.Count(coll)
}

// EJDB_EXPORT uint32_t ejdbupdate(EJCOLL *jcoll, bson *qobj, bson *orqobjs, int orqobjsnum, bson *hints, TCXSTR *log);
func (coll *EjColl) Update(query string, queries ...string) (int, *EjdbError) {
    query_bson := bson_from_json(query)
    defer C.bson_destroy(query_bson)

    orqueries := C.malloc(C.size_t(unsafe.Sizeof(C.bson{})) * C.size_t(len(queries)))
    defer C.free(orqueries)
    ptr_orqueries := (*[maxslice]C.bson)(orqueries)
    for i, q := range queries {
        bson := bson_from_json(q)
        (*ptr_orqueries)[i] = *bson
        defer C.bson_destroy(bson)
    }

    count := C.ejdbupdate((*C.struct_EJCOLL)(unsafe.Pointer(coll.ptr)), query_bson, (*C.bson)(orqueries), C.int(len(queries)), nil, nil)
    return int(count), coll.ejdb.check_error()
}

// Set index for JSON field in EJDB collection.
//
//  - Available index types:
//      - `JBIDXSTR` String index for JSON string values.
//      - `JBIDXISTR` Case insensitive string index for JSON string values.
//      - `JBIDXNUM` Index for JSON number values.
//      - `JBIDXARR` Token index for JSON arrays and string values.
//
//  - One JSON field can have several indexes for different types.
//
//  - Available index operations:
//      - `JBIDXDROP` Drop index of specified type.
//              - Eg: flag = JBIDXDROP | JBIDXNUM (Drop number index)
//      - `JBIDXDROPALL` Drop index for all types.
//      - `JBIDXREBLD` Rebuild index of specified type.
//      - `JBIDXOP` Optimize index of specified type. (Optimize the B+ tree index file)
//
//  Examples:
//      - Set index for JSON path `addressbook.number` for strings and numbers:
//          `ccoll.SetIndex("album.number", JBIDXSTR | JBIDXNUM)`
//      - Set index for array:
//          `ccoll.SetIndex("album.tags", JBIDXARR)`
//      - Rebuild previous index:
//          `ccoll.SetIndex("album.tags", JBIDXARR | JBIDXREBLD)`
func (coll *EjColl) SetIndex(ipath string, flags int) *EjdbError {
    c_ipath := C.CString(ipath)
    defer C.free(unsafe.Pointer(c_ipath))
    res := C.ejdbsetindex((*C.struct_EJCOLL)(unsafe.Pointer(coll.ptr)), c_ipath, C.int(flags))
    if res {
        return nil
    }
    return coll.ejdb.check_error()
}

// Begin transaction for EJDB collection.
func (coll *EjColl) BeginTransaction() *EjdbError {
    res := C.ejdbtranbegin((*C.struct_EJCOLL)(unsafe.Pointer(coll.ptr)))
    if res {
        return nil
    }
    return coll.ejdb.check_error()
}

// Commit transaction for EJDB collection.
func (coll *EjColl) CommitTransaction() *EjdbError {
    res := C.ejdbtrancommit((*C.struct_EJCOLL)(unsafe.Pointer(coll.ptr)))
    if res {
        return nil
    }
    return coll.ejdb.check_error()
}

// Abort transaction for EJDB collection.
func (coll *EjColl) AbortTransaction() *EjdbError {
    res := C.ejdbtranabort((*C.struct_EJCOLL)(unsafe.Pointer(coll.ptr)))
    if res {
        return nil
    }
    return coll.ejdb.check_error()
}

// Get current transaction status. Return true if a transaction is active, false otherwise.
func (coll *EjColl) IsTransactionActive() bool {
    var ret C.bool
    C.ejdbtranstatus((*C.struct_EJCOLL)(unsafe.Pointer(coll.ptr)), &ret)
    return bool(ret)
}

// Synchronize content of a EJDB collection database with the file on device. On success return true.
func (coll *EjColl) Sync() (bool, *EjdbError) {
    ret := C.ejdbsyncoll((*C.struct_EJCOLL)(unsafe.Pointer(coll.ptr)))
    if ret {
        return bool(ret), nil
    }
    return bool(ret), coll.ejdb.check_error()
}
