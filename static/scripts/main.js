window.addEventListener('load', () => {
  prepareDeleteButtons()
})

function prepareDeleteButtons() {
  const deleteForms = document.querySelectorAll('[action*="delete"]')
  deleteForms.forEach(form => {
    form.addEventListener('submit', function(e) {
      if (!confirm(this.dataset.deleteMessage)) e.preventDefault()
    })
  })
}