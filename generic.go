package main

import (
	"fmt"
	"math/rand"
	"time"
)

type Deck[C any] struct {
	cards []C
}

func (d *Deck[C]) AddCard(card C) {
	d.cards = append(d.cards, card)
}

func (d *Deck[C]) RandomCard() C {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	cardIdx := r.Intn(len(d.cards))
	return d.cards[cardIdx]
}

type PlayingCard struct {
	Suit string
	Rank string
}

func NewPlayingCard(suit string, rank string) *PlayingCard {
	return &PlayingCard{
		Suit: suit,
		Rank: rank,
	}
}

func NewPlayingCardDeck() *Deck[*PlayingCard] {
	suits := []string{"Diamonds", "Hearts", "Clubs", "Spades"}
	ranks := []string{"A", "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K"}

	deck := &Deck[*PlayingCard]{}
	for _, suit := range suits {
		for _, rank := range ranks {
			card := NewPlayingCard(suit, rank)
			deck.AddCard(card)
		}
	}

	return deck
}

func main() {
	deck := NewPlayingCardDeck()
	tradingCardDeck := NewTradingCardDeck()

	fmt.Printf("--- drawing playing card ---\n")
	playingCard := deck.RandomCard()
	fmt.Printf("drew card: %s\n", playingCard)
	// Code removed
	fmt.Printf("card suit: %s\n", playingCard.Suit)
	fmt.Printf("card rank: %s\n", playingCard.Rank)

	tradingCard := tradingCardDeck.RandomCard()
	fmt.Println(tradingCard.CollectableName)
}

type TradingCard struct {
	CollectableName string
}

func NewTradingCard(collectableName string) *TradingCard {
	return &TradingCard{CollectableName: collectableName}
}

func (t *TradingCard) String() string {
	return t.CollectableName
}

func NewTradingCardDeck() *Deck[*TradingCard] {
	collectableNames := []string{"Pikachu", "Charmander", "Bulbasaur", "Squirtle"}

	deck := &Deck[*TradingCard]{}
	for _, collectableName := range collectableNames {
		card := NewTradingCard(collectableName)
		deck.AddCard(card)
	}

	return deck
}
