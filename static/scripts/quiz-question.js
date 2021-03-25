window.addEventListener('load', () => {
  initAnswerInputPage()
})

initAnswerInputPage()

function initAnswerInputPage() {
  const addButton = document.querySelector('[data-add-answer]')
  console.log(addButton)
  if (!addButton) return
  addButton.addEventListener('click', addAnswer)
  const fieldset = document.querySelector('[data-answer-inputs]')
  fieldset.classList.remove('invisible')
}

function addAnswer() {
  const template = document.querySelector('[data-answer-template]')
  if (!template)
    return console.error('Template not found')

  const answerInputLabel = template.content.cloneNode(true)
  const label = answerInputLabel.querySelector('label')
  const index = document.querySelectorAll('[data-answer-inputs] input').length
  label.childNodes[0].nodeValue += ` ${index + 1}:`
  const input = answerInputLabel.querySelector('input')
  this.parentElement.insertBefore(answerInputLabel, this)
  input.focus()
}