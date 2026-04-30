#import "@preview/clean-dhbw:0.4.0": *
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
