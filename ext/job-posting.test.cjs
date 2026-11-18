const {test} = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const vm = require("node:vm");
const {extensionFixture} = require("./test-fixture.cjs");

function read(scripts, bodyText = "Visible page text for the fallback path.") {
  const engine = extensionFixture();
  engine.context.document = {
    querySelectorAll(selector) {
      if (selector !== 'script[type="application/ld+json"]') return [];
      return scripts.map((text) => ({textContent: text}));
    },
    body: {innerText: bodyText},
  };
  engine.context.location = {href: "https://jobs.example.invalid/role"};
  return vm.runInContext("readJobPage()", engine.context);
}

test("the page reader does not turn description HTML into page DOM", () => {
  const source = fs.readFileSync(`${__dirname}/background.js`, "utf8");
  const start = source.indexOf("function readJobPage()");
  const end = source.indexOf("// ── Page Text Extraction", start);
  assert.equal(source.slice(start, end).includes("innerHTML"), false);
  assert.equal(source.split("func: readJobPage").length - 1, 2);
});

test("top-level JobPosting fills the fields and does not keep the page text", () => {
  const page = read([JSON.stringify({
    "@context": "https://schema.org",
    "@type": "JobPosting",
    title: "Staff Engineer",
    description: "<p>Build <strong>APIs</strong>.</p><script>alert(1)</script><style>body{color:red}</style>",
    hiringOrganization: {"@type": "Organization", name: "Example"},
    jobLocation: {"@type": "Place", address: {addressLocality: "Remote", addressRegion: "CA", addressCountry: "US"}},
    baseSalary: {"@type": "MonetaryAmount", currency: "USD", value: {"@type": "QuantitativeValue", minValue: 120000, maxValue: 150000, unitText: "YEAR"}},
  })]);
  assert.equal(page.text, "");
  assert.equal(page.url, "https://jobs.example.invalid/role");
  assert.equal(page.jobPosting.title, "Staff Engineer");
  assert.equal(page.jobPosting.company, "Example");
  assert.equal(page.jobPosting.location, "Remote, CA, US");
  assert.equal(page.jobPosting.salary, "USD 120000\u2013150000 YEAR");
  assert.equal(page.jobPosting.description, "Build APIs.");
  assert.equal(page.jobPosting.description.includes("alert"), false);
});

test("JobPosting is found inside an array", () => {
  const page = read([JSON.stringify([
    {"@type": "Organization", name: "Not a job"},
    {"@type": "JobPosting", title: "Array role", description: "Array description"},
  ])]);
  assert.equal(page.jobPosting.title, "Array role");
  assert.equal(page.jobPosting.description, "Array description");
  assert.equal(page.text, "");
});

test("JobPosting is found inside @graph", () => {
  const page = read([JSON.stringify({
    "@context": "https://schema.org",
    "@graph": [{"@type": "JobPosting", title: "Graph role", description: "<div>Graph <em>work</em></div>"}],
  })]);
  assert.equal(page.jobPosting.title, "Graph role");
  assert.equal(page.jobPosting.description, "Graph work");
  assert.equal(page.text, "");
});

test("@type may be an array", () => {
  for (const type of [["JobPosting"], ["https://schema.org/JobPosting", "Thing"]]) {
    const page = read([JSON.stringify({"@type": type, title: "Typed role", description: "Typed description"})]);
    assert.equal(page.jobPosting.title, "Typed role");
    assert.equal(page.jobPosting.description, "Typed description");
  }
});

test("invalid JSON in one block does not hide a later JobPosting", () => {
  const page = read([
    "{not json",
    JSON.stringify({"@type": "JobPosting", title: "After the broken block", description: "Still read"}),
  ]);
  assert.equal(page.jobPosting.title, "After the broken block");
  assert.equal(page.jobPosting.description, "Still read");
  assert.equal(page.text, "");
});

test("a lone surrogate character reference does not fail the page read", () => {
  const page = read([JSON.stringify({
    "@type": "JobPosting",
    title: "Role",
    description: "Before &#55296; after &#xD800; done",
  })]);
  assert.equal(page.jobPosting.title, "Role");
  assert.equal(page.jobPosting.description, "Before after done");
  assert.equal(page.text, "");
});

test("a JobPosting without a title or description keeps the page text for the model", () => {
  const body = "Visible page text for the fallback path.";
  for (const posting of [
    {"@type": "JobPosting", title: "Title only", hiringOrganization: {name: "Example"}},
    {"@type": "JobPosting", description: "<p>Description without a title.</p>"},
  ]) {
    const page = read([JSON.stringify(posting)], body);
    assert.equal(page.jobPosting, null);
    assert.equal(page.text, body);
  }
});
