// Package storedfields packs the fields of one stored value into a byte record
// and reads them back. It keeps a stored record readable after its layout gains
// fields: a value written before those fields existed reads back with them zero,
// so a new field needs no migration and drops no stored data. Fields may only be
// appended - a layout that reorders, retypes, or drops a field changes the
// meaning of the records already stored. An instant is stored to the second.
package storedfields
