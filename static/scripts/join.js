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
  socket.addEventListener('open', () => {
  })
  socket.addEventListener('close', () => console.log('connection closed'))
  socket.addEventListener('message', e => {
  })
}
