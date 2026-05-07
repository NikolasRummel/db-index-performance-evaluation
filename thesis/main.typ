#import "@preview/clean-dhbw:0.5.0": *
#import "glossary.typ": glossary-entries, acrolist-entries

#register-glossary(acrolist-entries)

#show: clean-dhbw.with(
  
  title: "Analysis and Comparison of Database Index Structures",
  authors: (
    (name: "Nikolas Rummel", student-id: "7654321", course: "TINF23B6", course-of-studies: "Informatik"),
    // (name: "Juan Pérez", student-id: "1234567", course: "TIM21", course-of-studies: "Mobile Computer Science", company: (
    //   (name: "ABC S.L.", post-code: "08005", city: "Barcelona", country: "Spain")
    // )),
  ),
  type-of-thesis: "Studienarbeit", // Bachelorarbeit, Masterarbeit, Studienarbeit, Projektarbeit
  at-university: true, // if true the company name on the title page and the confidentiality statement are hidden
  city: "Karlsruhe",
  bibliography: bibliography("sources.bib"),
  date: datetime.today(),
  glossary: glossary-entries, // displays the glossary terms defined in "glossary.typ"
  language: "en", // en, de
  supervisor: (university: "Prof. Dr. Roland Schätzle"),
  university: "Duale Hochschule Baden-Württemberg",
  university-location: "Karlsruhe",
  university-short: "DHBW",
  // for more options check the package documentation (https://typst.app/universe/package/clean-dhbw)
  appendix: [
    = Acronyms
 
    #print-glossary(acrolist-entries)
  = AI Acknowledgement 
    The following table provides a transparent documentation of the artificial intelligence tools used during the creation of this thesis, in accordance with the guidelines for scientific work.

    #table(
      columns: (1fr, 3fr),
      inset: 10pt,
      stroke: 0.5pt + gray,
      fill: (column, row) => if row == 0 { luma(240) },
      align: (left, left),
      [*Tool*], [*Description of Use*],
      
      [Gemini (Web)], [
        - Understanding of basic concepts if questions arose during the research phase.
      ],
      
      [Gemini CLI], [
        - Assistance with code implementation and especially in debugging and optimizing the B-Tree and B+-Tree implementations.
        - Searching for grammar and spelling mistakes in thesis.
      ],

      [Claude (Web)], [
        - Helping to create the images for the thesis in typst with cetz, especially the B-Tree and B+-Tree images in @index.
      ],
      
      [NotebookLM], [
        - Summarization of topics in different sources, especially the original papers on B-Trees, B+-Trees and LSM-Trees, to get a better understanding of the concepts and to find relevant information for the thesis.
      ], 
    )
    
    #v(1em)
    I hereby declare that all AI-generated content was verified for factual accuracy and manually refined. No content was adopted without critical review.
  ]
)
#include "sections/introduction/introduction.typ"
#include "sections/fundamentals/dbms/dbms.typ"
#include "sections/fundamentals/dbms/storage.typ"
#include "sections/fundamentals/index/index.typ"
#include "sections/fundamentals/practice/practice.typ"
#include "sections/benchmark/benchmark.typ"
#include "sections/evaluation/evaluation.typ"
#include "sections/conclusion/conclusion.typ"
