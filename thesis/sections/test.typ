= Results and Analysis <evaluation>


#table(
columns: (1fr, 2fr),
inset: 10pt,
align: horizon,
table.header([Component], [Specification]),
[Processor], [Apple M1 Pro (10-Core)],
[Memory], [32 GB Unified Memory],
[Storage], [Internal NVMe SSD],
[Operating System], [macOS 16.3],
[File System], [APFS (Apple File System)],
[Go Runtime], [go1.22.x darwin/arm64],
) <tbl-specs>

#pagebreak()

= Conclusion <conclusion>

== Critical Reflection 
Benchmark like in sqlite -> 1 file for storing. Not optimized, fixed page length... 
Also compared LSM tree (real code) with B tree (own implementation) -> not really fair? 
More realistic workloads?



#pagebreak()

= Outlook <outlook>



