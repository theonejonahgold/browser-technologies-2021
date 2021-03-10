createScrollingImage()
createIntersectionObserver()

async function createScrollingImage() {
  const article = document.querySelector('article > div')
  const image = await fetch('speaker_css-js.svg').then(res => res.text())
  const imageContainer = document.createElement('div')
  imageContainer.classList.add('image-container')
  imageContainer.classList.add('stick')
  imageContainer.innerHTML = image
  article.prepend(imageContainer)
  const speakerSvgEL = document.querySelector(".image-container.stick>svg");
  speakerSvgEL.classList.add("speakerHtml");
  const staticImages = document.querySelectorAll('article section img')
  staticImages.forEach(img => img.classList.add('hidden'))
}

function createIntersectionObserver() {
  const observer = new IntersectionObserver(intersectingHandler, { threshold: 0.8 })
  const sections = document.querySelectorAll('article div section')
  sections.forEach(observer.observe.bind(observer))
}

function intersectingHandler(entries) {
  const speakerSvgEL = document.querySelector(".image-container.stick>svg");
  if (entries.some((entry) => entry.target.dataset.type === "html")) { //check html
      speakerSvgEL.classList.remove("speakerJs", "speakerCss");
      speakerSvgEL.classList.add("speakerHtml");
  } else if (entries.some(entry => entry.target.dataset.type === "css")) { // check css
      speakerSvgEL.classList.remove("speakerHtml", "speakerJs");
      speakerSvgEL.classList.add("speakerCss");
  } else if (entries.some(entry => entry.target.dataset.type === "js")) { // check js
      speakerSvgEL.classList.remove("speakerHtml", "speakerCss");
      speakerSvgEL.classList.add("speakerJs");
  }
}