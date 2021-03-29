# Goldhoot

![Badge showing that the project is MIT licensed](https://img.shields.io/github/license/theonejonahgold/goldhoot?style=flat-square) ![Badge showing amount of open issues](https://img.shields.io/github/issues/theonejonahgold/goldhoot?style=flat-square)

[Link to website](https://browser-tech-goldhoot.herokuapp.com/)

Goldhoot is a quiz platform to the likes of [Kahoot](https://kahoot.com). Create quizzes, invite people, have fun!

## Index

- [Features](#features)
- [Enhancements](#enhancements)
- [Wireflow](#wireflow)
- [Testing](#testing)
	- [Chosen browsers](#chosen-browsers)

## Features

- [x] Create quiz
	- [x] Publish quiz
	- [x] Share quiz with link
	- [x] Start a quiz
	- [x] Proceed to next question
	- [x] See results of all questions in quiz
- [x] Create question
	- [x] Change question title
	- [x] Delete question
	- [x] Change question position in quiz
- [x] Create answer
	- [x] Edit answer
	- [x] Delete answer
- [x] Profile page (became homepage)
- [x] Join quiz
	- [x] Answer questions
	- [x] See results for current question
	- [x] Continue to the next question

## The three layers of progressive enhancement

1. All basic content is displayed in semantic HTML. The quiz flow is kept up-to-date with frequent page refreshes, which might be a bit jarring, but all moments where user input is required are not automatically refreshed to make the experience more bearable.
2. With CSS I bring more hierarchy to the content. The content is usable by itself, but the CSS just gives the content the extra push it needs to be fully usable.
3. With JS I take interactivity to a whole new level. Using Web Sockets and template tags I have made sure to streamline the quiz flow, creating and editing questions, and more!

A list of all the enhancements I've put in place are found below.

### List of all enhancements

- CSS Grid layouts, built upon working flexbox layouts.
- A proper countdown with CSS animations, instead of the numbers "3", "2" and "1" statically displayed on the screen.
- Deletion confirmations with `confirm()`.
- "Add answer" creating a new answer input instead of submitting the form and thus refreshing the page.
- Web Sockets when hosting or joining a quiz, streamlining the experience and making it snappier for all parties.

## Wireflow

![Picture of wireflow](docs/wireflow.png)

[PDF version](docs/wireflow.pdf)

## Testing

### Chosen browsers

- Safari 14 on macOS Big Sur 11.2.3
- Chrome 89 on macOS Big Sur 11.2.3
- Safari 14 on iPhone with iOS 14.4.1
- Firefox 80.1.1 on Google Pixel 5 with Android 11

### Feature testing

<!-- 
	FEATURES
	1. QUIZ AANMAKEN
	2. VRAAG TOEVOEGEN
	3. ANTWOORDEN TOEVOEGEN
	4. QUIZ HOSTEN
	5. QUIZ JOINEN

	TEST SCENARIOS
	1. NO JS
	2. NO JS EN CSS
	3. NO COOKIES
-->

<!-- Add a nice poster image here at the end of the week, showing off your shiny frontend 📸 -->

<!-- How about a section that describes how to install this project? 🤓 -->

<!-- How about a license here? 📜 (or is it a licence?) 🤷 -->
