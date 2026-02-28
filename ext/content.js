// Content script: extracts page text when requested by popup
chrome.runtime.onMessage.addListener((request, _sender, sendResponse) => {
  if (request.action === "extractPageText") {
    const text = document.body.innerText.trim().substring(0, 50000);
    sendResponse({ text, url: location.href });
  }
  return true;
});
