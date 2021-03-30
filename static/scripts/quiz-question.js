window.addEventListener('load', () => {
  initAnswerInputPage()
})

function initAnswerInputPage() {
  const addButton = document.querySelector('[data-add-answer]')
  if (!addButton) return
  addButton.setAttribute('type', 'button')
  addButton.addEventListener('click', addAnswer)
  const currentInputs = document.querySelectorAll('[data-answer-inputs] input')
  currentInputs.forEach(input =>
    input.addEventListener('keypress', addAnswerBasedOnKeypress)
  )
  const nameInput = document.querySelector('input[name="title"]')
  nameInput.addEventListener('keypress', addAnswerBasedOnKeypress)
}

function addAnswer() {
  const lastInput = document.querySelector(
    '[data-answer-inputs] label:last-of-type input'
  )
  if (lastInput.value === '') return void lastInput.focus()
  const template = document.querySelector('[data-answer-template]')
  if (!template) return console.error('Template not found')
  const content = 'content' in template ? template.content : template
  const answerInputLabel = content.cloneNode(true)
  const label = answerInputLabel.querySelector('label')
  const index = document.querySelectorAll('[data-answer-inputs] input').length
  label.childNodes[0].nodeValue += ` ${index + 1}:`
  const input = answerInputLabel.querySelector('input')
  this.parentElement.insertBefore(answerInputLabel, this)
  input.focus()
  input.addEventListener('keypress', addAnswerBasedOnKeypress)
}

function addAnswerBasedOnKeypress(e) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    addAnswer.call(document.querySelector('[data-add-answer]'))
  }
}
