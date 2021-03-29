/*global splitQuery*/

window.addEventListener('load', () => {
  prepareContent()
  joinWS()
})

function prepareContent() {
  const contentContainer = document.querySelector('[data-content]')
  contentContainer.innerHTML = '<p>Loading...</p>'
}

function joinWS() {
  const host = window.location.hostname
  const port = window.location.port
  const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const entries = splitQuery()
  const socket = new WebSocket(
    `${wsProtocol}//${host}:${port}/app/ws?sessid=${entries.sessid}&session=${entries.session}`,
    'join'
  )
  socket.addEventListener('close', () => console.log('connection closed'))
  socket.addEventListener('message', e => {
    const message = JSON.parse(e.data)
    switch (message.type) {
      case 'joined':
        renderToDOM(renderWaitingRoom(message))
        break
      case 'countdown':
        renderToDOM(renderCountdown(message))
        break
      case 'open':
        renderToDOM(renderAnswer(message))
        updateCountdown(message)
        document
          .querySelector('[data-content] [data-answer-form]')
          .addEventListener('submit', function (e) {
            e.preventDefault()
            const formData = new FormData(this)
            const answer = formData.get('answer')
            const sessid = formData.get('sessid')
            const newMessage = {
              type: 'answer',
              sessid,
              answer,
              quizid: message.quizid,
            }
            socket.send(JSON.stringify(newMessage))
          })
        break
      case 'confirmed':
        renderToDOM(renderAnswered(message))
        break
      case 'results':
        renderToDOM(renderResult(message))
        break
      case 'finished':
        socket.close()
        window.location.href = `${window.location.origin}/app/quiz/${message.quizid}/results?sessid=${message.sessid}`
        break
    }
  })
}

function renderWaitingRoom({ quiz, host }) {
  const content = queryTemplateContent('waiting')
  const heading = content.querySelector('[data-quiz-name]')
  heading.textContent = quiz.name
  const text = content.querySelector('[data-quiz-host]')
  text.textContent = text.textContent.replace('{}', host)
  return content
}

function renderCountdown({ question, last }) {
  const content = queryTemplateContent('countdown')
  const heading = content.querySelector('header h1')
  if (last) heading.textContent = 'Last question'
  else heading.textContent = 'Next question'
  const title = content.querySelector('[data-question-title]')
  title.textContent = question.title
  return content
}

function renderAnswer({ question, sessid, timeLimit }) {
  const content = queryTemplateContent('answer')
  const title = content.querySelector('[data-question-title]')
  title.textContent = question.title
  const sessionInput = content.querySelector('[data-sessid]')
  sessionInput.value = sessid
  const fieldset = content.querySelector('[data-answer-form] fieldset')
  const answerTemplate = content.querySelector('[data-answer-input]')
  question.answers.forEach(answer => {
    const answerInput = answerTemplate.content.cloneNode(true)
    const input = answerInput.querySelector('input')
    const label = answerInput.querySelector('label')
    input.id = answer.title
    input.value = answer._id
    label.setAttribute('for', answer.title)
    label.textContent = answer.title
    fieldset.appendChild(answerInput)
  })
  const timer = content.querySelector('[data-timer]')
  if (timeLimit == 0) {
    timer.remove()
    return content
  }
  timer.textContent = `${timeLimit} seconds left`
  return content
}

function renderAnswered({ answer, question }) {
  const content = queryTemplateContent('answered')
  const title = content.querySelector('[data-question-title]')
  title.textContent = question.title
  const chosenAnswer = content.querySelector('[data-chosen-answer]')
  chosenAnswer.textContent = chosenAnswer.textContent.replace('{}', answer)
  return content
}

function renderResult({ question, participantAmount, last }) {
  const content = queryTemplateContent('result')
  const heading = content.querySelector('header h1')
  if (last) heading.textContent = 'Last question results'
  else heading.textContent = 'Question results'
  const title = content.querySelector('[data-question-title]')
  title.textContent = question.title
  const answerTemplate = content.querySelector('[data-answer-result]')
  const main = content.querySelector('main')
  question.answers.forEach(answer => {
    const answerResult = answerTemplate.content.cloneNode(true)
    const label = answerResult.querySelector('label')
    label.childNodes[0].nodeValue = label.childNodes[0].nodeValue.replace(
      '{answer}',
      answer.title
    )
    label.childNodes[0].nodeValue = label.childNodes[0].nodeValue.replace(
      '{amount}',
      answer.participants.length
    )
    const meter = answerResult.querySelector('meter')
    meter.max = participantAmount
    meter.value = answer.participants.length
    meter.textContent = meter.textContent.replace(
      '{amount}',
      answer.participants.length
    )
    meter.textContent = meter.textContent.replace('{answer}', answer.title)
    main.appendChild(answerResult)
  })
  return content
}

function renderToDOM(node) {
  const container = document.querySelector('[data-content]')
  container.innerHTML = ''
  container.appendChild(node)
}

function queryTemplateContent(name) {
  const template = document.querySelector(`[data-${name}]`)
  const content = template.content.cloneNode(true)
  return content
}

function updateCountdown({ timeLimit }) {
  let currentTime = timeLimit
  const interval = setInterval(() => {
    const timer = document.querySelector('[data-timer]')
    if (!timer || currentTime === 0) {
      clearInterval(interval)
      return
    }
    timer.textContent = `${--currentTime} seconds left`
  }, 1000)
}
