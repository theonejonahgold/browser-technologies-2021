window.addEventListener('load', () => {
  prepareDeleteButtons()
})

function prepareDeleteButtons() {
  const deleteForms = document.querySelectorAll('[action*="delete"]')
  deleteForms.forEach(form => {
    form.addEventListener('submit', function (e) {
      if (!confirm(this.dataset.deleteMessage)) e.preventDefault()
    })
  })
}

function splitQuery() {
  const query = window.location.search
  const entries = query
    .slice(1)
    .split('&')
    .map(query => query.split('='))
    .reduce((acc, curr) => ({ ...acc, [curr[0]]: curr[1] }), {})
  return entries
}
