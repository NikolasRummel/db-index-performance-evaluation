#import "@preview/clean-dhbw:0.4.0": gls

#import "@preview/cetz:0.4.2"

== Storage and Storage Management <storage>
To understand how a #gls("DBMS") handles data, it is essential to distinguish between the physical storage media and the storage management layer inside the #gls("DBMS"). While storage refers to the hardware devices (e.g. a #gls("SSD")) and their physical characteristics, storage management is the software component of the #gls("DBMS"), that provides an abstraction between the logical data model and the physical hardware.

=== Physical Storage Media <physical_storage>
The choice of storage media directly impacts the performance, durability, and cost of a database system. Modern systems typically navigate a trade-off between speed and persistence within the memory hierarchy.

#figure(
  caption: [The Memory Hierarchy of Computers @scottLatency],
  cetz.canvas({
    import cetz.draw: *

    let volatile-fill = blue.lighten(90%)
    let persistent-fill = green.lighten(90%)
    let text-size = 8pt

    // --- 1. THE PYRAMID SEGMENTS ---
    
    // Level 6 (Top): Registers
    line((0, 6), (0.66, 5), (-0.66, 5), close: true, fill: volatile-fill, stroke: 0.5pt)
    content((0, 5.4), text(size: text-size)[Registers])

    // Level 5: Cache
    line((-0.66, 5), (0.66, 5), (1.33, 4), (-1.33, 4), close: true, fill: volatile-fill, stroke: 0.5pt)
    content((0, 4.5), text(size: text-size)[Caches])

    // Level 4: RAM
    line((-1.33, 4), (1.33, 4), (2, 3), (-2, 3), close: true, fill: volatile-fill, stroke: 0.5pt)
    content((0, 3.5), text(size: text-size)[Main Memory (RAM)])

    // --- Boundary Line (The "Storage Gap") ---
    line((-5.5, 3), (5.5, 3), stroke: (thickness: 1.5pt, paint: gray))

    // Level 3: SSD
    line((-2, 3), (2, 3), (2.66, 2), (-2.66, 2), close: true, fill: persistent-fill, stroke: 0.5pt)
    content((0, 2.5), text(size: text-size)[SSD (Flash)])

    // Level 2: HDD
    line((-2.66, 2), (2.66, 2), (3.33, 1), (-3.33, 1), close: true, fill: persistent-fill, stroke: 0.5pt)
    content((0, 1.5), text(size: text-size)[HDD (Magnetic)])

    // Level 1: Remote
    line((-3.33, 1), (3.33, 1), (4, 0), (-4, 0), close: true, fill: persistent-fill, stroke: 0.5pt)
    content((0, 0.5), text(size: text-size)[Cloud / Remote])

    // --- 2. LEFT SIDE: ACCESS & COST (Pushed further left to -5.5) ---
    line((-5.8, 0), (-5.8, 6), mark: (start: "stealth", end: "stealth", scale: 0.6))
    content((-6.0, 4.5), anchor: "east", text(size: 7pt)[Faster Access,\ Higher Cost])
    content((-6.0, 1.5), anchor: "east", text(size: 7pt)[Slower Access,\ Lower Cost])

    // --- 3. RIGHT SIDE: LABELS & BRACKETS (Pushed further right) ---
    // Latency estimates (Placed relative to the right slope)
    content((1.0, 5.5), anchor: "west", text(size: 7pt)[~1 ns])
    content((1.7, 4.5), anchor: "west", text(size: 7pt)[~4 ns])
    content((2.4, 3.5), anchor: "west", text(size: 7pt)[~100 ns])
    content((3.1, 2.5), anchor: "west", text(size: 7pt)[~16 #sym.mu s])
    content((3.8, 1.5), anchor: "west", text(size: 7pt)[~2 ms])
    content((4.5, 0.5), anchor: "west", text(size: 7pt)[~>50 ms])

    // Primary Storage Bracket (Pushed to 6.2)
    line((5.8, 6), (6.1, 6), (6.1, 3.1), (5.8, 3.1), stroke: 0.5pt)
    content((6.3, 4.5), anchor: "west", text(size: 8pt, weight: "bold")[Primary \ Storage])
    
    // Secondary Storage Bracket
    line((5.8, 2.9), (6.1, 2.9), (6.1, 0), (5.8, 0), stroke: 0.5pt)
    content((6.3, 1.5), anchor: "west", text(size: 8pt, weight: "bold")[Secondary \ Storage])

    // --- 4. BOTTOM: CAPACITY ---
    line((-4, -0.6), (4, -0.6), mark: (start: "stealth", end: "stealth", scale: 0.6))
    content((0, -0.9), text(size: 8pt)[Storage Capacity])
  })
) <memory-pyramid>

So one might think that the best choice would be to use the fastest storage media available, but this is not the case. The amout of registers, caches (and the main memory) is very limited and expensive and not big enough for a typical database usecase @elmasri2016[p. 542]. Moreover, the data is volatile, meaning after some power outage or system crash, the data would be lost @elmasri2016[p. 542], what is not want for a database. Therefore, there is a need to use persistent storage media like a #gls("HDD") and nowadays especially #gls("SSD")s.  

In the following, the characteristics of these storage media will be described to further understand indexes and their implementation in a #gls("DBMS"). 

==== Hard Disk Drives (HDD)
#gls("HDD")s are electromechanical storage devices that use spinning magnetic disks to store data. They have been the dominant form of secondary storage for decades due to their large capacity and relatively low cost @elmasri2016[p. 547]. However, they have slower access times and lower throughput compared to newer storage technologies like #gls("SSD"). 
A #gls("HDD") has multiple discs which hold data and a read/write head that moves across the surface of the discs to access data. 

#figure(
  image("../../../assets/hdd.png", width: 50%),
  caption: [A schematic of a hard disk drive],
) <hdd-schematic>

On each of these discs, data is organized in concentric circles called tracks, which are further divided into #gls("Sector")s. A #gls("Sector") is the smallest addressable unit on the physical disk, traditionally 512 bytes or 4 KB in size @elmasri2016[p. 549]. 

The read/write head moves to the appropriate track and #gls("Sector") to access data.
The performance of a #gls("HDD") is influenced by factors such as seek time (the time it takes for the head to move to the correct track), rotational latency (the time it takes for the desired #gls("Sector") to rotate under the head), and transfer rate (the speed at which data can be read or written once the head is in position) @elmasri2016[p. 547]. In total, the access time can be calculated as follows:
$$
$ T_(a c c e s s) = T_(s e e k) + T_(r o t) + T_(t r a n s f e r) $
$$

Data is then read or written in #gls("Block")s, which are typically 4KB in size @elmasri2016[p. 547]. The performance of a #gls("HDD") can be significantly affected by the access pattern, as sequential access minimizes seek time and rotational latency, while random access lead to increased latency due to the need for the head to move around the disk. 

/*
#figure(
  block(
    fill: luma(250),
    inset: 15pt,
    radius: 4pt,
    stroke: 0.5pt + luma(200),
    width: 100%,
    [
      #set text(size: 9pt)
      #align(left)[
        *Example Calculation: Impact of Access Patterns on I/O Performance* \
        To quantify the impact of mechanical latency, consider a drive with an average seek time $T_(s e e k) = 10"ms"$ and rotational latency $T_(r o t) = 4"ms"$. The time required to retrieve $n=100$ non-contiguous vs. contiguous #gls("Block")s is:

        1. *Random Access:*
           $ t_(r a n d) = n dot (T_(s e e k) + T_(r o t)) = 100 dot (10"ms" + 4"ms") = bold(1.4"s") $

        2. *Sequential Access:*
           $ t_(s e q) = T_(s e e k) + T_(r o t) + (n dot T_(t r a n s f e r)) approx bold(0.02"s") $
      ]

      #v(1em) // Add some vertical space before the chart

      // --- Performance Visualization ---
      #cetz.canvas(length: 1cm, {
        import cetz.draw: *
        
        let b_height = 0.6
        
        // Random Bar (Tiny)
        rect((0, 1.2), (0.1, 1.2 + b_height), fill: red.lighten(80%), name: "rand")
        content("rand.west", [Random], anchor: "east", padding: .2, size: 8pt)
        content("rand.east", [ ~150 Operations Per Second], anchor: "west", size: 7pt)
        
        // Sequential Bar (Large)
        rect((0, 0.2), (5, 0.2 + b_height), fill: green.lighten(80%), name: "seq")
        content("seq.west", [Sequential], anchor: "east", padding: .2, size: 8pt)
        content("seq.east", [ ~10.000+ Operations Per Second], anchor: "west", size: 7pt)
        
        // X-Axis
        line((0, 0), (6, 0), mark: (end: "stealth", scale: 0.5))
        content((3, -0.3), [Throughput / Operations per Second], size: 7pt, fill: gray.darken(30%))
      })
    ]
  ),
  caption: [Comparison of I/O Performance Between Random and Sequential Access Patterns TODO: fix numbers.],
  kind: "calculation",
  supplement: [Calculation]
) <calc-io-comparison>
*/

==== Solid State Drives (SSD) <ssd_chapter>
On the other side, #gls("SSD") use semiconductor-based #gls("NAND") flash memory. Because they lack mechanical components, the physical constraints of seek time ($T_(s e e k)$) and rotational latency ($T_(r o t)$) are eliminated. Instead, performance is determined by electrical signal propagation and the efficiency of the internal controller.
#figure(
  image("../../../assets/ssd.png", width: 70%),
  caption: [A schematic of a solid state drive @ssd], 
) <ssd-schematic>

Inside of the actual storage, the data is written in flash cells, which are the smallest unit of storage in a #gls("SSD"), where bits in transistors are stored @os[p. 1]. Those are organized in #gls("Page")s with a typical size of 4kb, which then are grouped into #gls("Block")s of 128KB to 256 KB size @os[p. 1-2]. The performance of a #gls("SSD") is influenced by factors such as the type of #gls("NAND") flash type, the efficiency of the internal controller, and the #gls("Wearleveling") of the flash memory @os[p. 1]. 

#figure(
  image("../../../assets/ssd2.png", width: 70%),
  caption: [A simple flash chip @os[p. 2]], 
) <flash-chip>

To now read a physical #gls("Page"), we have a constant access time even though the data can be stored anywhere. This is called random access @os[p. 3] and is the key advantage of #gls("SSD") over #gls("HDD"). However, writing to a #gls("SSD") is more complex. Due to the nature of flash memory, data cannot be overwritten in place. Instead, an entire #gls("Block") must be erased before new data can be written, which leads to increased latency for write operations and can cause performance degradation over time as the drive fills up @os[p. 3]. To mitigate this issue, #gls("SSD") use techniques like #gls("Wearleveling") and #gls("gc") to manage the flash memory and maintain performance @os[p. 3-4]. 
However, #gls("SSD", plural:true) also provide better performance for sequential writes compared to random ones and therefore are very suitable for write throughput @kleppmann[p. 79]. This is because sequential patterns allow the internal controller to utilize parallel #gls("NAND") channels and minimize the overhead of #gls("gc") and #gls("writeamplification"). 

==== Buffer Management 
In order to speed up data acces, the goal of a #gls("DBMS") is to keep as much data as possible in main memory, since access to main memory is much faster than access to secondary storage (see @memory-pyramid).

The `Buffer Manager` is responsible for smartly managing the most important data in the main memory to speed up query performance. In a #gls("DBMS"), the `Buffer Manager` holds a pool of database #gls("Page")s in main memory, which are used to cache data from disk @elmasri2016[p. 557]. These logical database #gls("Page")s serve as the atomic unit of transfer between software and storage, abstracting the underlying hardware specifics we saw in @physical_storage. Since main memory is limited, the `Buffer Manager` must decide which database #gls("Page") to keep and which to evict when new data is required. This is achieved through buffer replacement policies, which determine eviction based on factors such as recency of access, frequency of use, and the specific cost of reloading a #gls("Page") from the underlying physical storage @elmasri2016[p. 559]

Common buffer replacement policies include:
- *#gls("LRU")*: Evicts the #gls("Page") that has not been accessed for the longest time.
- *#gls("MRU")*: Evicts the #gls("Page") that was accessed most recently
- *#gls("FIFO")*: Evicts the #gls("Page") that has been in the buffer pool the longest.
- *Clock*: Evicts the #gls("Page") that has been accessed least recently, but uses a circular buffer and a "use" bit to track #gls("Page") usage.

In addition to these policies, in practice, many #gls("DBMS") implement more sophisticated algorithms that take into account the specific workload and access patterns to optimize performance. However, for the purpose of this thesis, we will focus on the most common ones.

==== Data Organization: The Slotted Page Model
As previously mentioned, the #gls("DBMS") interacts with the storage layer in fixed-size database #gls("Page")s. However, the data stored within these #gls("Page")s, such as database rows or index entries, often has a variable size. Names for instance don't have the same length, and to not waste space, the #gls("DBMS") needs to be able to manage variable-length records within a fixed-size #gls("Page").

To manage this efficiently, the Slotted Page Model is used @silberschatz2020database[pp. 592-594].
In this model, a #gls("Page") is divided into three main sections:
1. *Header:* Contains metadata such as the #gls("Page") ID, the number of slots, and a pointer to the start of free space.
2. *Slot Directory:* An array of pointers (offsets) located at the front of the #gls("Page") that track the starting location of each record.
3. *Data Area:* The actual records, which are typically inserted from the end of the #gls("Page") moving backwards toward the header.

#figure(
  rect(width: 60%, height: 4cm, stroke: 0.5pt, fill: luma(250))[
    #set align(center + horizon)
    #grid(
      rows: (1fr, 1fr, 3fr),
      rect(width: 100%, fill: gray.lighten(50%))[Page Header],
      rect(width: 100%, fill: gray.lighten(80%))[Slot Directory (Pointers)],
      rect(width: 100%)[Free Space / Records (Slotted Data)]
    )
  ],
  caption: [The Slotted Page Architecture used for internal Page organization.],
) <fig-slotted-page>

This architecture is essential for efficiently managing variable-length records within fixed-size #gls("Page")s, allowing the #gls("DBMS") to optimize storage utilization and access patterns while maintaining the necessary metadata for record management. 

=== Summary
Understanding both the physical limitations of storage media and the logical mechanisms of storage management is fundamental to database design. While the `Buffer Manager` attempts to mask disk latency by caching data, the underlying organization of data into #gls("Page")s remains a critical factor in performance. 

In the following chapter, we will build upon these concepts to explore how index structures utilize this #gls("Page")-based storage management to provide logarithmic search performance, what makes it much more efficient then full-table scans.