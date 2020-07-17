# Pivot

Provides a way to synchronize multiple instances of a server

https://storage.googleapis.com/pub-tools-public-publication-data/pdf/65b514eda12d025585183a641b5a9e096a3c4be5.pdf

> The CAP theorem [Bre12] says that you can only have two of the three desirable properties of:
• C: Consistency, which we can think of as serializability for this discussion;
• A: 100% availability, for both reads and updates;
• P: tolerance to network partitions.

>This leads to three kinds of systems: CA, CP and AP, based on what letter you leave out. Note that you are
not entitled to 2 of 3, and many systems have zero or one of the properties.

This distribution sistem is CP, if a node is not available it should not accept writes or deletes.

