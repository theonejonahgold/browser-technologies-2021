/*global splitQuery*/

window.addEventListener('load', () => {
  prepareContent()
  hostWS()
})

function prepareContent() {
  const contentContainer = document.querySelector('[data-content]')
  contentContainer.innerHTML = '<p>Loading...</p>'
}

function hostWS() {
  const host = window.location.hostname
  const port = window.location.port
  const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const entries = splitQuery()
  const socket = new WebSocket(
    `${wsProtocol}//${host}:${port}/app/ws?sessid=${entries.sessid}&session=${entries.session}`,
    'host'
  )
  socket.addEventListener('close', () => console.log('connection closed'))
  socket.addEventListener('message', e => {
    const message = JSON.parse(e.data)
    switch (message.type) {
      case 'joined':
        renderToDOM(renderWaitingRoom(message))
        document
          .querySelector('[data-content] [data-start]')
          .addEventListener('click', () => {
            const newMessage = {
              type: 'start',
              sessid: message.sessid,
              quizid: message.quizid,
            }
            socket.send(JSON.stringify(newMessage))
          })
        break
      case 'participant':
        updateWaitingRoom(message)
        break
      case 'countdown':
        renderToDOM(renderCountdown(message))
        break
      case 'open':
        renderToDOM(renderAnswer(message))
        updateCountdown(message)
        break
      case 'answered':
        updateAnsweredAmt(message)
        break
      case 'results':
        renderToDOM(renderResult(message))
        document
          .querySelector('[data-content] [data-next]')
          .addEventListener('click', () => {
            const newMessage = {
              type: 'next',
              sessid: message.sessid,
              quizid: message.quizid,
            }
            socket.send(JSON.stringify(newMessage))
          })
        break
      case 'finished':
        socket.close()
        window.location.href = `${window.location.origin}/app/quiz/${message.quizid}/results?sessid=${message.sessid}`
        break
    }
  })
}

function renderWaitingRoom({ quiz }) {
  const content = queryTemplateContent('waiting')
  const heading = content.querySelector('[data-quiz-name]')
  heading.textContent = quiz.name
  const joinedText = content.querySelector('[data-participants]')
  joinedText.textContent = `0 people have joined`
  const codeInput = content.querySelector('input')
  codeInput.value = quiz.code
  return content
}

function updateWaitingRoom({ amount }) {
  const joinedText = document.querySelector(
    '[data-content] [data-participants]'
  )
  joinedText.textContent = `${amount} people have joined`
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

function renderAnswer({ question, participantAmount, timeLimit }) {
  const content = queryTemplateContent('answer')
  const title = content.querySelector('[data-question-title]')
  title.textContent = question.title
  const progress = content.querySelector('[data-progress]')
  progress.textContent = `0 of ${participantAmount} participants answered`
  const timer = content.querySelector('[data-timer]')
  timer.textContent = `People have ${timeLimit} seconds left to answer`
  return content
}

function updateAnsweredAmt({ amount, participantAmount }) {
  const progress = document.querySelector('[data-content] [data-progress]')
  progress.textContent = `${amount} of ${participantAmount} participants answered`
}

function renderResult({ question, participantAmount, last }) {
  const content = queryTemplateContent('result')
  const heading = content.querySelector('header h1')
  if (last) heading.textContent = 'Last question results'
  else heading.textContent = 'Question results'
  const title = content.querySelector('[data-question-title]')
  title.textContent = question.title
  const nextButton = content.querySelector('[data-next]')
  if (last) nextButton.textContent = 'Finish quiz'
  else nextButton.textContent = 'Next question'
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
    main.insertBefore(answerResult, nextButton)
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
    timer.textContent = `People have ${--currentTime} seconds left to answer`
  }, 1000)
}
