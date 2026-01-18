<div align = "center">
  <img alt = "project logo" src = "https://github.com/jamesread/TheDarkStation/blob/main/logo.svg" width = "128" />
  <h1>The Dark Station</h1>

  A relaxing, single-player exploration puzzle game. Navigate an abandoned space station, restore power to generators, unlock sealed doors, and find your way to the next deck.

[![Maturity Badge](https://img.shields.io/badge/maturity-Beta-yellow)](#none)
[![Discord](https://img.shields.io/discord/846737624960860180?label=Discord%20Server)](https://discord.gg/jhYWWpNJ3v)
[![Go Report Card](https://goreportcard.com/badge/github.com/jamesread/TheDarkStation)](https://goreportcard.com/report/github.com/jamesread/TheDarkStation)
[![License](https://img.shields.io/github/license/jamesread/TheDarkStation)](LICENSE)

</div>

## Screenshot

```
Deck 3

You are in Damaged Engineering

Inventory: Map, Red Key, Batteries x3

Generators: #1 2/3, #2 POWERED

- ?: Show hint

>
```

## Features

- **Procedurally generated** space station layouts using BSP trees
- **Field of view system** with line-of-sight blocking
- **Locked doors** requiring keycards found throughout the station
- **Generator power system** requiring batteries to unlock exits
- **Environmental hazards** - vacuum breaches, coolant leaks, electrical faults
- **Interactive furniture** with hidden items and atmospheric descriptions
- **Dynamic viewport** that scales to terminal size
- **Relaxing gameplay** - no timers, no health bars, just exploration and logic puzzles

## Installation

### From Release

Download the latest release for your platform from the [Releases](https://github.com/jamesread/TheDarkStation/releases) page.

### From Source

```bash
git clone https://github.com/jamesread/TheDarkStation.git
cd TheDarkStation
go build -o darkstation main.go
./darkstation
```

## Usage

```bash
# Start a new game
./darkstation

# Start at a specific level (for testing)
./darkstation -level 5
```

### Controls

- **N/S/E/W** or **Arrow Keys** - Move in cardinal directions
- **H/J/K/L** - Vim-style movement
- **?** - Show a hint
- **quit** or **q** - Exit the game

## How to Play

1. Explore the station by moving between rooms
2. Find keycards to unlock sealed doors
3. Collect batteries to power generators
4. Clear environmental hazards by finding control panels or repair items
5. Search furniture for hidden items
6. Find the lift to advance to the next deck

## License

This project is licensed under the AGPL-3.0 License - see the [LICENSE](LICENSE) file for details.
