# Database Index Benchmark Suite - Complete Architecture & Interaction Summary

**Date:** February 25, 2026  
**Project:** B-Tree, B+ Tree, LSM-Tree Benchmark Harness  
**Language:** Go  
**Location:** `/Users/nikolasrummel/dev/studium/Studienarbeit/src`

---

## Table of Contents

1. [Project Overview](#project-overview)
2. [File Structure & Components](#file-structure--components)
3. [Component Interactions](#component-interactions)
4. [Data Flow](#data-flow)
5. [Key Design Patterns](#key-design-patterns)
6. [Execution Flow](#execution-flow)

---

## Project Overview

**Purpose:** Benchmark three disk-based index structures to compare insertion speed, point-query latency, range-scan performance, and write amplification under varying workloads.

**Three Index Implementations:**
- **B-Tree** (classic): Data stored in all nodes; no leaf chaining; in-order traversal for range scans
- **B+ Tree**: Data only in leaves; leaf chaining for fast sequential access; internal nodes route only
- **LSM-Tree**: Write-optimized; in-memory memtable + disk-based SSTables; compaction threads

**Testing Approach:**
- Sequential, random, skewed distributions
- Variable dataset sizes: 10K–2M keys
- Range scans with different selectivities
- Point read/write operations
- Time-series latency tracking

---

## File Structure & Components

### Root Level

```
src/
├── main.go              # Entry point; orchestrates benchmark execution
├── main2.go             # Functional test harness for quick verification
├── go.mod               # Go module definition
├── go.sum               # Dependency checksums
└── benchmark/           # Benchmark runner & telemetry
    ├── benchmark.go     # Core benchmark orchestrator
    ├── workload.go      # Workload generation (Sequential, Random, Skewed)
    └── plots.go         # Result plotting & CSV output
```

### Storage Layer: `dbms/pager/`

**File:** `pager.go`

**Purpose:** Unified abstraction for disk I/O across all index implementations.

**Key Components:**
- **`Pager` struct**: Manages file I/O and caching
  - `file`: Underlying OS file handle (`.bt`, `.bpt`, `.lsm`)
  - `cache`: LRU page cache (default 256 pages ≈ 1 MB)
  - `pageCount`: Total pages ever allocated
  
- **Page Size:** 4096 bytes (OS page aligned)

**Public API:**
```go
Open(path, cacheSize) (*Pager, error)     // Open/create file
Read(pageID) (*Page, error)               // Get page (cached or disk)
Write(pageID, *Page) error                // Write page + cache
Allocate() (pageID, error)                // Allocate new page
PageCount() uint64                        // Total pages
Close() error                             // Cleanup
```

**Internal Structure:**
- **LRU Cache**: Doubly-linked list + hash map
- **Disk Layout**: Page 0 = metadata, Page 1 = index header, Pages 2+ = tree nodes
- **Offset Calculation:** `offset = pageID × 4096`

---

### Index Interface: `dbms/index/`

**File:** `index.go`

**Purpose:** Common interface for all index implementations.

```go
type Index interface {
    Insert(key int64, value []byte) error
    Get(key int64) ([]byte, error)
    Delete(key int64) error
    Range(start, end int64) (Iterator, error)
    Close() error
}

type Iterator interface {
    Next() bool
    Key() int64
    Value() []byte
    Error() error
    Close() error
}
```

**Implementations:**
1. B-Tree (`btree/`)
2. B+ Tree (`bptree/`)
3. LSM-Tree (`lsm/`)
4. List Index (`listindex/`) - reference baseline
5. Shared Tree Engine (`shared/`)

---

### Page Layout: `dbms/index/btpage/`

**File:** `page.go`

**Purpose:** Unified on-disk page format for B-Tree and B+ Tree (shared engine model).

**Page Layout (4096 bytes):**
```
[0]        1 byte    Page type (TypeInternal=0 / TypeLeaf=1)
[1-2]      2 bytes   Number of cells (numCells)
[3-4]      2 bytes   Cell content start (free space pointer)
[5-8]      4 bytes   Rightmost child ID (internal nodes)
[9-12]     4 bytes   Next leaf ID (B+ tree leaves; else InvalidPage)
[13+]      Variable  Cell pointer array (grows down) + cell data (grows up)
```

**Key Functions:**
```go
InitPage(p, type)                  // Initialize page
NumCells(p) int                    // Get cell count
CellPtr(p, i) uint16              // Get cell offset
AllocCell(p, size) int            // Allocate space for new cell
FreeSpace(p, n) int               // Get available space
SetNumCells(p, n)                 // Update cell count
SetNextLeaf(p, id) / NextLeaf(p)  // B+ tree leaf chaining
SetRightmost(p, id) / Rightmost() // Child pointer for internal nodes
```

**Cell Format** (variable by implementation):
- **B-Tree internal:** `[leftChild:4][key:8][value:var]`
- **B-Tree leaf:** Same format
- **B+ Tree internal:** `[leftChild:4][key:8]` (no value)
- **B+ Tree leaf:** `[key:8][valueLen:2][value:var]`

---

### Shared Tree Engine: `dbms/index/shared/`

**File:** `tree.go`

**Purpose:** Generic B-tree/B+ tree implementation parameterized by `NodeAccessor` interface.

**Key Types:**

**`NodeAccessor` Interface** (implemented per tree type):
```go
type NodeAccessor interface {
    CellSize(isLeaf bool, value []byte) int
    ReadCell(p, i, isLeaf) (key, value, leftChild)
    WriteCell(p, off, key, value, leftChild, isLeaf)
    OverwriteValue(p, i, newVal, isLeaf)
    CopyUpLeaves() bool              // B+ = true, B-Tree = false
    LinkLeaves(left, right, newID, oldNext)
}

type Tree struct {
    Pg     *Pager
    RootID uint32
    Acc    NodeAccessor               // Behavior adapter
}
```

**Core Operations:**

1. **`Get(key)`**: Binary search from root
   - B-Tree: Check internal nodes for exact match, descend if not found
   - B+ Tree: Descend to leaf only, then search

2. **`Insert(key, value)`**: Recursive insertion with split handling
   - Find position (left == right child when not found)
   - Insert if space available, else split
   - Promote median key to parent (copy-up for B+, push-up for B-Tree)
   - Propagate splits up tree

3. **`Range(start, end)`**: Initiate range iterator
   - Navigate to first leaf containing `start`
   - Return iterator

4. **`FindLeaf(key)`**: Navigate to leaf (B+ Tree only)
   - Descend internal nodes until leaf reached

---

### B-Tree Implementation: `dbms/index/btree/`

**File:** `btree.go`

**Purpose:** Classic B-tree with data in all nodes.

**`BTreeAcc` Struct** (NodeAccessor implementation):
```go
type BTreeAcc struct{}

func (BTreeAcc) CopyUpLeaves() bool { return false }  // Push-up semantics
func (BTreeAcc) LinkLeaves(_, _, _, _) {}              // No chaining
func (BTreeAcc) CellSize(isLeaf, value) int {
    return 4 + 8 + 2 + len(value)  // leftChild + key + valueLen + data
}
```

**Key Behavior:**
- **Search**: Can terminate in internal nodes if key exists there
- **Split**: Median key is removed from source and placed in parent only
- **Range**: Uses stack-based in-order traversal (no leaf chain)

**`RangeIterator` Struct**:
```go
type frame struct {
    id          uint64
    idx         int
    subtreeDone bool
}

type RangeIterator struct {
    tree  *BTree
    end   int64
    stack []frame          // Explicit tree traversal stack
    k, v  int64, []byte
}
```

**Range Scan Algorithm**:
1. Initialize stack with path from root to first key ≥ start
2. `Next()`: Advance through in-order traversal
   - Return key from internal/leaf node
   - When exhausted, backtrack to parent
   - Continue to right sibling (via rightmost pointer)

---

### B+ Tree Implementation: `dbms/index/bptree/`

**File:** `bptree.go`

**Purpose:** B+ tree with data only in leaves and leaf chaining.

**`BPTreeAcc` Struct**:
```go
type BPTreeAcc struct{}

func (BPTreeAcc) CopyUpLeaves() bool { return true }   // Copy-up semantics
func (BPTreeAcc) LinkLeaves(left, right, newID, oldNext) {
    SetNextLeaf(left, newID)      // left → newID
    SetNextLeaf(right, oldNext)   // newID → old right
}
func (BPTreeAcc) CellSize(isLeaf, value) int {
    if isLeaf {
        return 8 + 2 + len(value)  // key + valueLen + data
    }
    return 4 + 8                   // leftChild + key (no value)
}
```

**Key Behavior:**
- **Search**: Always descends to leaf; internal nodes are routing only
- **Split**: Median key copied up to parent; remains in right leaf
- **Range**: Follows leaf chain `nextLeaf` pointers (O(1) per key)

**`RangeIterator` Struct**:
```go
type RangeIterator struct {
    tree   *BPTree
    end    int64
    leafID uint64    // Current leaf (single pointer, no stack)
    idx    int
    k, v   int64, []byte
}
```

**Range Scan Algorithm**:
1. `Range(start, end)`: Find leaf containing `start`
2. `Next()`: Sequential scan
   - Return keys from current leaf while `idx < NumCells`
   - When exhausted, follow `nextLeaf` pointer
   - Stop when `leafID == InvalidPage` or `key > end`

---

### LSM-Tree Implementation: `dbms/index/lsm/`

**File:** `lsm.go`

**Purpose:** Write-optimized index with background compaction.

**Architecture:**
- **Memtable**: In-memory red-black tree or skip list (configurable)
- **Levels**: L0, L1, L2, ... on disk as SSTable files
- **Compaction**: Background thread merges L0 → L1, L1 → L2, etc.

**Key Components:**
```go
type LSMTree struct {
    memtable    *SkipList            // In-memory buffer
    levels      map[int][]*SSTable   // L0, L1, L2, ...
    compactor   *Compactor           // Background thread
    mtx         sync.RWMutex         // Concurrency control
}

type SSTable struct {
    DataBlocks  [][]*KeyValue
    BloomFilter *BloomFilter         // Probabilistic key test
    IndexBlock  map[int64]BlockRef
}
```

**Operations:**
- **Insert**: Write to memtable (in-memory) → return immediately
- **Get**: Check memtable → search levels (newest first) → use bloom filters
- **Range**: Merge iterators from all levels
- **Background**: Flush memtable to L0 when full; compact levels when threshold met

---

### Benchmark Framework: `benchmark/`

**Files:**
- `benchmark.go` - Core orchestrator
- `workload.go` - Key distribution generators
- `plots.go` - CSV output & graph generation

#### `benchmark.go`

**`Config` Struct**:
```go
type Config struct {
    Ops             int       // Operations per benchmark
    PreloadSize     int       // Initial dataset size
    KeySpace        int64     // Total key range
    DatasetSizes    []int     // Sizes to test: 10K, 50K, 100K, ...
    RangeWidths     []int64   // Range scan sizes: 10, 100, 1K, 10K
    WriteWindowSize int       // Batch size for write tracking
    TotalWriteOps   int       // Total write operations
    OutDir          string    // Results output directory
    Seed            int64     // RNG seed for reproducibility
    CachePages      int       // LRU cache size
}
```

**`Runner` Struct**:
```go
type Runner struct {
    cfg     Config
    engines []EngineFactory    // B-Tree, B+ Tree, LSM instances
    results []Result           // Collected metrics
    baseDir string             // Temp storage for databases
}
```

**Main Experiments:**
1. **Insert Throughput**: Sequential, random, skewed distributions
2. **Point Read Throughput**: Random lookups in pre-loaded data
3. **Range Scan**: Variable scan widths (10–10,000 keys)
4. **Write Over Time**: Continuous writes with latency tracking

#### `workload.go`

**Key Generators**:
```go
type KeyGenerator interface {
    Next() int64
}

// Implementations:
type Sequential struct{ current, max int64 }
type Random struct{ rng *rand.Rand }
type Skewed struct{ rng *rand.Rand }  // Zipfian distribution
type Uniform struct{ rng *rand.Rand }
```

#### `plots.go`

**Functions**:
- `GeneratePlots(csvPath, outDir)`: Read results CSV → create PNG graphs
- CSV columns: `engine`, `experiment`, `label`, `ops_per_sec`

---

### Entry Points

#### `main.go`

**Flow:**
1. Create output directories (`results/data/`, `results/plots/`)
2. Check if results exist; if not, run benchmark
3. Call `benchmark.GeneratePlots()` to create visualization

**Engine Factory:**
```go
engines := []EngineFactory{
    {"btree", func(path) => btree.Open(path, cachePages)},
    {"bptree", func(path) => bptree.Open(path, cachePages)},
    {"lsm_pebble", func(path) => lsm.Open(path)},
}
```

#### `main2.go`

**Purpose:** Quick functional test (NOT benchmarking).

**Test Scenarios:**
1. Insert out-of-order keys
2. Update existing keys
3. Point lookups
4. Range scans with edge cases:
   - Full range [min, max]
   - Partial ranges [60, 130]
   - Out-of-bounds [200, 300]
   - Single-point [50, 50]
5. Stress test: 500 sequential inserts → verify last key in range

**Output:**
```
--- Testing B-Tree ---
1. Basic Functional Tests...
2. Range Scan Edge Cases...
Range [25, 175] OK (7 items)
...
3. Split & Continuity Stress Test...
Stress test passed.
```

---

## Component Interactions

### Interaction Diagram

```
┌─────────────────────────────────────────────────────────┐
│                    main.go                              │
│           (Entry point + Configuration)                 │
└─────────────────────────────────────────────────────────┘
                            │
                            ▼
        ┌────────────────────────────────────────┐
        │  benchmark.Runner.RunAll()             │
        │  (Orchestrate experiments)             │
        └────────────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        ▼                   ▼                   ▼
   runInsertThroughput  runPointRead      runRangeScan
        │                   │                   │
        └───────────────────┼───────────────────┘
                            │
                            ▼
            ┌───────────────────────────────┐
            │  EngineFactory.NewFunc()      │
            │  (Instantiate 3 indices)      │
            └───────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        ▼                   ▼                   ▼
    btree.Open()        bptree.Open()       lsm.Open()
    (Classic B-T)       (B+ Tree)          (LSM-Tree)
        │                   │                   │
        └───────────────────┼───────────────────┘
                            │
                            ▼
            ┌───────────────────────────────┐
            │  shared.Tree (generic engine) │
            │  + NodeAccessor (per type)    │
            └───────────────────────────────┘
                            │
                            ▼
            ┌───────────────────────────────┐
            │  pager.Pager                  │
            │  (Page cache + disk I/O)      │
            └───────────────────────────────┘
                            │
                            ▼
            ┌───────────────────────────────┐
            │  OS File I/O                  │
            │  (.bt, .bpt, .lsm files)      │
            └───────────────────────────────┘
```

### Key Interaction Patterns

#### 1. **Index Creation & Initialization**

```
main.go
  └─> benchmark.NewRunner(engines)
        └─> btree.Open(path, cachePages)
              └─> pager.Open(path+".bt", cachePages)
                    └─> Create file header (page 0)
                    └─> Allocate root node (page 1)
                    └─> Initialize LRU cache
```

#### 2. **Insertion Flow**

```
benchmark.runInsertThroughput()
  └─> for each key in KeyGenerator:
        └─> index.Insert(key, value)
              └─> shared.Tree.Insert()
                    └─> insertRec(nodeID, key, value)
                          └─> pager.Read(nodeID) [cached or disk read]
                          └─> findPosition(key)
                          └─> if space available: doInsert()
                          └─> else: splitNode()
                                └─> allocate newPageID
                                └─> pager.Write() [updates cache + disk]
                                └─> promote median key to parent
```

#### 3. **Range Scan Flow (B+ Tree)**

```
benchmark.runRangeScan()
  └─> index.Range(start, end)
        └─> bptree.Range(start, end)
              └─> FindLeaf(start)
                    └─> binary descent from root to first leaf
                    └─> pager.Read() at each level
              └─> initialize RangeIterator(leafID, idx)

  └─> for it.Next():
        └─> pager.Read(leafID)
        └─> read cells [idx, n)
        └─> if exhausted: leafID = NextLeaf(page)
        └─> return key/value pairs
```

#### 4. **Point Query Flow**

```
benchmark.runPointRead()
  └─> for each random key in preloaded dataset:
        └─> index.Get(key)
              └─> shared.Tree.Get(key)
                    └─> curr = rootID
                    └─> while curr != null:
                          └─> page = pager.Read(curr)
                          └─> find key in cells
                          └─> if B+: descend to leaf only
                          └─> if B-Tree: can match in internal node
                          └─> return value or descend to child
```

#### 5. **Caching Behavior**

```
pager.Read(pageID)
  ├─> Check LRU cache
  │     └─> if hit: mark as recently used, return
  │     └─> if miss: read from disk
  │
  └─> readPageFromDisk(pageID)
        └─> offset = pageID × 4096
        └─> os.File.ReadAt(buf, offset)
        └─> cache.put(pageID, page)
        └─> return page

pager.Write(pageID, page)
  ├─> cache.put(pageID, page)  [update in-memory]
  └─> writePageToDisk(pageID, page)
        └─> os.File.WriteAt(page[:], offset)
```

---

## Data Flow

### Insertion Data Flow (with Caching)

```
User Input: key=42, value="hello"
          │
          ▼
    benchmark.runInsertThroughput()
          │
          ▼
    index.Insert(42, "hello")
          │
          ├─────────────────────────────────────┐
          │ B-Tree / B+ Tree / LSM               │
          │ (different behaviors)                │
          │
          └─ shared.Tree.Insert(42, "hello")   │
               │                                │
               ▼                                │
          insertRec(rootID, 42, "hello")       │
               │                                │
               ▼                                │
          pager.Read(rootID)                   │
               │                                │
               ├─ LRU Cache Hit? ──> page data │
               │                                │
               └─ LRU Cache Miss?               │
                    │                           │
                    ▼                           │
               readPageFromDisk()               │
                    │                           │
                    ▼                           │
               os.File.ReadAt()                 │
                    │                           │
                    ▼                           │
               store in LRU cache               │
               
          Binary search for position in page
               │
               ├─ Space available?              │
               │    └─ doInsert() + pager.Write()
               │
               └─ Space full?                   │
                    └─ splitNode() + promote
                         └─ insertRec(parent)  │ (recurse)
```

### Query Data Flow

```
User Input: get(42)
          │
          ▼
    index.Get(42)
          │
          ├─ LSM-Tree?
          │    └─ Check memtable ──> if found, return
          │    └─ Search L0 ──> L1 ──> L2
          │         (use bloom filters to skip)
          │
          └─ B-Tree / B+ Tree?
               │
               ▼
          shared.Tree.Get(42)
               │
               ▼
          Navigate from root
               │
               ├─ B-Tree: Check every node for exact match
               │
               └─ B+ Tree: Descend to leaf only
               
               Each level:
                    │
                    ▼
               pager.Read(pageID)
                    │
                    ├─ LRU Hit? ──> immediate
                    └─ Miss? ──> disk read + cache
```

### Range Scan Data Flow (B+ Tree Advantage)

```
User Input: range(100, 500)
          │
          ▼
    index.Range(100, 500)
          │
          ▼
    bptree.Range(100, 500)
          │
          ├─ Navigate to first leaf containing key ≥ 100
          │    └─ pager.Read() at each level
          │    └─ root → internal → internal → leaf
          │
          ▼
    Initialize RangeIterator(leafID, startIdx)
          │
    Loop: it.Next()
          │
          ├─ Scan current leaf [startIdx, endIdx)
          │    └─ pager.Read(leafID) ──> return keys/values
          │    └─ increment idx
          │
          ├─ If idx reaches end of leaf:
          │    └─ leafID = NextLeaf(page) ──> O(1)!
          │    └─ reset idx = 0
          │
          └─ Stop when leafID == InvalidPage or key > 500

*** vs B-Tree Range Scan ***

btree.Range(100, 500)
          │
          ├─ Initialize explicit stack with path to first key ≥ 100
          │
    Loop: it.Next()
          │
          ├─ Return current key from stack
          │
          ├─ Advance to next key (in-order):
          │    └─ Move to right sibling (via stack traversal)
          │    └─ May need to backtrack multiple levels
          │    └─ Re-descent into left subtree
          │
          └─ Stop when key > 500

⇒ B+ Tree: ~O(N) simple sequential reads
⇒ B-Tree: ~O(N * log depth) with tree traversal overhead
```

---

## Key Design Patterns

### 1. **Shared Engine Pattern**

**Problem:** B-Tree and B+ Tree share 90% of logic (search, insert, split, rebalance).

**Solution:** Extract generic `shared.Tree` parameterized by `NodeAccessor` interface.

```go
type shared.Tree struct {
    Pg     *Pager
    RootID uint32
    Acc    NodeAccessor  // Behavior adapter
}

// BTree-specific behavior
type BTreeAcc struct{}
func (BTreeAcc) CopyUpLeaves() bool { return false }  // Push-up
func (BTreeAcc) LinkLeaves(_, _, _, _) {}              // No chaining

// B+ Tree-specific behavior
type BPTreeAcc struct{}
func (BPTreeAcc) CopyUpLeaves() bool { return true }   // Copy-up
func (BPTreeAcc) LinkLeaves(l, r, newID, oldNext) {
    SetNextLeaf(l, newID)
    SetNextLeaf(r, oldNext)
}
```

**Benefit:** Single split/merge implementation; behavior differs only by accessor methods.

---

### 2. **Pager Abstraction Layer**

**Problem:** Each index needs to manage disk I/O, page allocation, caching.

**Solution:** Single `Pager` layer used by all indices.

```go
// Unified interface
pager.Open(path, cacheSize)
pager.Read(pageID)    // Cached or disk read
pager.Write(pageID, page)
pager.Allocate()      // New page ID
```

**Benefit:** Consistent cache behavior, fair comparison, easy to swap I/O strategies.

---

### 3. **Page-Aligned Memory Layout**

**Problem:** Disk I/O is optimized for page-sized chunks (OS level); random offsets are slow.

**Solution:** Fixed 4 KB page size matching OS page cache.

```go
const PageSize = 4096

// Every node fits exactly one page
// No overflow, no split storage
type Page [PageSize]byte

// Predictable offset calculation
offset = pageID × 4096
```

**Benefit:** Single disk read per page; predictable latency; no fragmentation.

---

### 4. **Factory Pattern for Engines**

**Problem:** Benchmark needs to instantiate B-Tree, B+ Tree, LSM dynamically.

**Solution:** `EngineFactory` with custom `NewFunc`.

```go
type EngineFactory struct {
    Name    string
    NewFunc func(path string) (index.Index, error)
}

engines := []EngineFactory{
    {Name: "btree", NewFunc: btree.Open},
    {Name: "bptree", NewFunc: bptree.Open},
    {Name: "lsm", NewFunc: lsm.Open},
}

for _, e := range engines {
    idx := e.NewFunc(basePath)
    // Run benchmark against idx
}
```

**Benefit:** Easy to add new index types; no code duplication in benchmark loop.

---

### 5. **Iterator Pattern for Range Scans**

**Problem:** B-Tree and B+ Tree have completely different range scan algorithms.

**Solution:** Common `Iterator` interface; implementations differ.

```go
type Iterator interface {
    Next() bool
    Key() int64
    Value() []byte
    Close() error
}

// B-Tree: Stack-based in-order traversal
type btree.RangeIterator struct {
    stack []frame      // Explicit tree path
    subtreeDone bool
}

// B+ Tree: Leaf chain walk
type bptree.RangeIterator struct {
    leafID uint64
    idx    int
}

// Benchmark code (identical for both):
it, _ := index.Range(start, end)
for it.Next() {
    process(it.Key(), it.Value())
}
```

**Benefit:** Benchmark code is engine-agnostic; each engine optimizes internally.

---

### 6. **Reproducible Randomization**

**Problem:** Benchmark results must be reproducible; random keys differ on each run.

**Solution:** Seeded `rand.Rand` from config.

```go
type Config struct {
    Seed int64  // e.g., 42
}

// In workload generation
rng := rand.New(rand.NewSource(cfg.Seed))

// Same seed ⇒ same sequence every run
```

**Benefit:** Results are deterministic; easier to debug; compare across runs.

---

## Execution Flow

### Complete Benchmark Execution

```
1. main.go::main()
   ├─ Create output directories
   ├─ Check if results exist
   │
   └─ If NOT exist:
      │
      └─ runBenchmark()
           │
           ├─ Create cfg := DefaultConfig()
           │  (Ops=100K, PreloadSize=500K, etc.)
           │
           ├─ Instantiate 3 engines:
           │  ├─ btree.Open("results/data/btree", cachePages)
           │  ├─ bptree.Open("results/data/bptree", cachePages)
           │  └─ lsm.Open("results/data/lsm")
           │
           ├─ runner := NewRunner(cfg, engines, baseDir)
           │
           └─ runner.RunAll()  // Main orchestrator
                │
                ├─ Experiment 1: runInsertThroughput()
                │  ├─ For each distribution (seq, random, skewed):
                │  │  ├─ Clear databases
                │  │  ├─ Create workload (KeyGenerator)
                │  │  ├─ For each engine:
                │  │  │  ├─ index.Insert(key, value) × Ops times
                │  │  │  ├─ Measure latency
                │  │  │  └─ Compute ops/sec
                │  │  └─ Append Result
                │  │
                │  └─ Append results to runner.results
                │
                ├─ Experiment 2: runPointRead()
                │  ├─ Preload dataset (500K keys)
                │  ├─ For each engine:
                │  │  ├─ Generate 100K random lookups
                │  │  ├─ Measure ops/sec
                │  │  └─ Append Result
                │  │
                │  └─ Append results to runner.results
                │
                ├─ Experiment 3: runRangeScan()
                │  ├─ For each range width (10, 100, 1K, 10K):
                │  │  ├─ For each dataset size:
                │  │  │  ├─ For each engine:
                │  │  │  │  ├─ Generate 100K range queries
                │  │  │  │  ├─ Measure ops/sec
                │  │  │  │  └─ Append Result
                │  │
                │  └─ Append results to runner.results
                │
                ├─ Experiment 4: runWriteOverTime()
                │  ├─ Continuous writes; sample every WriteWindowSize ops
                │  ├─ Track latency degradation over time
                │  │
                │  └─ Append results to runner.results
                │
                └─ writeCSV()
                   └─ Output "results/results.csv"
                      (engine, experiment, label, ops_per_sec)

2. benchmark.GeneratePlots()
   ├─ Read results.csv
   ├─ Parse data per (engine, experiment)
   ├─ Generate PNG graphs via gnuplot/matplotlib
   │  ├─ throughput_insert_sequential.png
   │  ├─ throughput_pointread.png
   │  ├─ latency_range_scan.png
   │  └─ ... (one per experiment)
   │
   └─ Save to "results/plots/"

3. main.go::main() returns
   ├─ Print "Finished!"
   └─ All data saved
```

### Quick Test Execution (main2.go)

```
1. main2.go::main()
   │
   ├─ os.Remove("test.bt")    // Clean up old files
   ├─ os.Remove("test.bpt")
   │
   ├─ btree.Open("test", cachePages=10)
   │  └─ pager.Open("test.bt", 10)
   │     └─ Create fresh database
   │
   ├─ runTest(btree):
   │  ├─ Insert 7 keys out of order (100, 50, 150, ...)
   │  ├─ Update key 75
   │  ├─ Verify point lookups
   │  │  ├─ Get(75) ⇒ "updated-75" ✓
   │  │  ├─ Get(999) ⇒ nil ✓
   │  │
   │  ├─ Test range scans:
   │  │  ├─ Range(25, 175) ⇒ 7 items ✓
   │  │  ├─ Range(60, 130) ⇒ 3 items ✓
   │  │  ├─ Range(200, 300) ⇒ 0 items ✓
   │  │  ├─ Range(0, 10) ⇒ 0 items ✓
   │  │  └─ Range(50, 50) ⇒ 1 item ✓
   │  │
   │  └─ Stress test: Insert 500 sequential keys (1000–1499)
   │     └─ Range(1490, 1510) ⇒ last key is 1499 ✓
   │
   ├─ bptree.Open("test", cachePages=10)
   │  └─ Repeat runTest(bptree)
   │
   └─ Print "Stress test passed."
```

---

## Summary Table

| Component | Purpose | Key Abstraction | Used By |
|-----------|---------|-----------------|---------|
| **pager** | Disk I/O + caching | `Page[4096]byte`, `Pager` | All indices |
| **index** | Common interface | `Index`, `Iterator` | Benchmark harness |
| **btpage** | Unified page layout | Page header fields | B-Tree, B+ Tree |
| **shared** | Generic B-tree engine | `Tree`, `NodeAccessor` | B-Tree, B+ Tree |
| **btree** | Classic B-tree | `BTreeAcc` (push-up, no chaining) | Benchmark |
| **bptree** | B+ tree | `BPTreeAcc` (copy-up, leaf chain) | Benchmark |
| **lsm** | LSM-tree | Memtable + Levels + Compaction | Benchmark |
| **benchmark** | Orchestrate tests | `Runner`, `Config`, `EngineFactory` | main.go |
| **main** | Entry point | Config loading, results generation | User |

---

## Key Files & Lines of Code

```
dbms/pager/pager.go        ~250 LOC   LRU cache + disk I/O
dbms/index/index.go        ~20 LOC    Interface definitions
dbms/index/btpage/page.go  ~100 LOC   Page layout helpers
dbms/index/shared/tree.go  ~450 LOC   Shared B-tree engine
dbms/index/btree/btree.go  ~180 LOC   B-tree adapter + iterator
dbms/index/bptree/bptree.go ~190 LOC   B+ tree adapter + iterator
dbms/index/lsm/lsm.go      ~300 LOC   LSM-tree implementation
benchmark/benchmark.go     ~360 LOC   Experiment orchestration
benchmark/workload.go      ~100 LOC   Key generation
benchmark/plots.go         ~80 LOC    Results output
main.go                    ~100 LOC   Entry point
main2.go                   ~105 LOC   Functional test harness
```

**Total: ~2000+ lines of Go code**

---

## Conclusion

This is a well-architected benchmark suite demonstrating:

1. **Abstraction layers** (Pager, shared engine) for code reuse
2. **Parameterized behavior** (NodeAccessor) for variant implementations
3. **Fair comparison** (same cache, page layout, seed)
4. **Reproducibility** (seeded RNG, configuration)
5. **Comprehensive testing** (4+ experiments, multiple distributions)
6. **Clean interfaces** (Iterator, Index) for extensibility

The interaction model is **hierarchical**: Benchmark → Engines → Shared Engine → Pager → Disk, with each layer providing abstraction and reusability.
