# C* Time Series

Maintainer: dg@HailoOSS.com

This readme is brief on the basis that we're recreating this functionality
using CQL3 within `Om` - so it will be explained in more detail there.

The basic idea is that we store **entities**, ordered by time, within C*.
As a user, I should be able to say:

 | Give me a list of "things" between date/time X and date/time Y

The way we store this in C* is that we store entire serialised entities (JSON)
within column values, and we leverage the column name to give us time ordering,
using a bespoke hand-crafted column name with a time component and then a unique
ID, eg: `1387072974-7375`.

To avoid rows growing too big (a C* anti-pattern), we choose a row key that
relates to a "bucket" of time. This is configurable, and is the **single most
important thing you need to choose** if using this package. If you choose a day,
then all the _things_ that have a time within a single day will be stored in 
one row. If you had 1M things happening on a day, this would be a problem.


## Integration tests

Setup to run against `boxen` (hence hard-coded port 19160). You need to create
the following schema definitions:

```
create keyspace testing;
use testing;
create column family TestNarrowRow;
create column family TestNarrowRowIndex;
create column family TestWideRow;
create column family TestWideRowIndex;
create column family TestPrimeRow;
create column family TestPrimeRowIndex;
create column family TestNoInterval;
create column family TestNoIntervalIndex;
create column family TestLargeInterval;
create column family TestLargeIntervalIndex;
```

Then run this:

```
go test -tags=integration -v .
```
