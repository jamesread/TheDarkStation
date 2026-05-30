package main

import (
	"flag"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/leonelquinteros/gotext"

	engineinput "darkstation/pkg/engine/input"
	"darkstation/pkg/game/deck"
	"darkstation/pkg/game/devtools"
	"darkstation/pkg/game/gameplay"
	gamemenu "darkstation/pkg/game/menu"
	"darkstation/pkg/game/renderer"
	ebitenRenderer "darkstation/pkg/game/renderer/ebiten"
	"darkstation/pkg/game/state"
)

// Version information set via ldflags during build
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func initGettext() {
	// Load embedded .mo file directly
	mo := gotext.NewMo()
	mo.Parse(defaultMO)

	// Create a locale and add the embedded translations
	locale := gotext.NewLocale("", "en_GB.utf8")
	locale.AddTranslator("default", mo)

	// Set the locale as the global storage
	gotext.SetLocales([]*gotext.Locale{locale})
}

func main() {
	startLevel := flag.Int("level", 1, "starting level/deck number (for developer testing)")
	flag.Parse()

	// Check for LEVEL environment variable (takes precedence over flag)
	if envLevel := os.Getenv("LEVEL"); envLevel != "" {
		if parsedLevel, err := strconv.Atoi(envLevel); err == nil && parsedLevel > 0 {
			*startLevel = parsedLevel
		}
	}

	initGettext()
	rand.Seed(time.Now().UnixNano())

	// Set version information for renderers
	renderer.SetVersion(version, commit, date)
	log.Printf("Starting TheDarkCastle (built %s, commit: %s)", renderer.BuildLabel, commit)

	// Initialize the Ebiten renderer
	ebitRenderer := ebitenRenderer.New()
	ebitRenderer.SetLongUseAdvancer(gameplay.AdvanceLongUseIfActive)
	ebitRenderer.SetHazardClearAdvancer(gameplay.AdvanceHazardClearIfActive)
	ebitRenderer.SetHazardTourAdvancer(gameplay.AdvanceHazardTourIfActive)
	renderer.SetRenderer(ebitRenderer)
	renderer.Init()

	// Open the window first, then run the menu inside the game loop
	log.Println("Opening main window...")

	// Run a single game loop that handles both menu and game
	if err := ebitRenderer.RunWithGameLoop(func() {
		for {
			// Run the main menu (this blocks until user makes a selection)
			menuAction := runMainMenuInLoop()

			// Build the game based on menu selection
			var g *state.Game
			switch menuAction {
			case gamemenu.MainMenuActionGenerate:
				// Start normal game mode (level from -level flag or LEVEL env)
				g = gameplay.BuildGame(*startLevel)
			case gamemenu.MainMenuActionDebug:
				// Open Developer map (normally opened on F9)
				g = state.NewGame()
				devtools.SwitchToDevMap(g)
			case gamemenu.MainMenuActionQuit:
				// Quit (should have been handled in RunMainMenu, but just in case)
				os.Exit(0)
			default:
				// Fallback: start normal game (level from -level flag or LEVEL env)
				g = gameplay.BuildGame(*startLevel)
			}

			// Reset QuitToTitle flag
			g.QuitToTitle = false

			// Now run the actual game loop
			for {
				mainLoop(g)
				// Check if we should quit to title
				if g.QuitToTitle {
					g.ResetAllProgress()
					break
				}
			}

			// If we broke out due to QuitToTitle, the outer loop will continue
			// and show the main menu again
		}
	}); err != nil {
		log.Printf("Failed to open main window: %v", err)
		os.Exit(1)
	}
}

// runMainMenuInLoop runs the main menu inside the Ebiten game loop
// This allows the menu to render and receive input properly
func runMainMenuInLoop() gamemenu.MainMenuAction {
	// Create a minimal game state for the menu (needed for rendering)
	g := state.NewGame()

	for {
		handler := gamemenu.NewMainMenuHandler()
		items := handler.GetMenuItems()
		gamemenu.RunMenu(g, items, handler)

		if handler.ShouldQuit() {
			os.Exit(0)
		}

		action := handler.GetSelectedAction()

		// Handle bindings menu as a sub-menu
		if action == gamemenu.MainMenuActionBindings {
			gameplay.RunBindingsMenu(g, true) // true = from main menu, shows "Back" option
			// After bindings menu closes, continue the main menu loop
			continue
		}

		// For other actions, return to let the caller handle them
		return action
	}
}

func mainLoop(g *state.Game) {
	if g.QuitToTitle {
		return
	}

	// Completion screen: stats, credits, then return to title (non-blocking so animations run).
	if g.GameComplete {
		intent := engineinput.Intent{Action: engineinput.ActionNone}
		if pending, ok := renderer.TryGetIntent(); ok {
			intent = pending
		}
		gameplay.ProcessCompletionInput(g, intent)
		if g.QuitToTitle {
			return
		}
		gameplay.UpdateCompletionSequence(g)
		if g.QuitToTitle {
			return
		}
		renderer.RenderFrame(g)
		time.Sleep(16 * time.Millisecond)
		return
	}

	renderer.Clear()

	if g.CurrentCell == nil || g.Grid == nil {
		return
	}

	renderer.ClearCalloutsIfMoved(g.CurrentCell.Row, g.CurrentCell.Col)
	renderer.ShowRoomEntryIfNew(g.CurrentCell.Row, g.CurrentCell.Col, g.CurrentCell.Name)

	if g.ExitAnimating {
		elapsed := time.Now().UnixMilli() - g.ExitAnimStartTime
		const exitAnimDuration = 2000 // 2 seconds (matches drawExitAnimation)
		if elapsed >= exitAnimDuration {
			g.ExitAnimating = false
			gameplay.AdvanceLevel(g)
		}
	} else if g.CurrentCell.ExitCell {
		// Final deck: lift has no destination; game complete (GDD §10.2, §11)
		if deck.IsFinalDeck(g.Level) {
			gameplay.TriggerGameComplete(g)
		} else if !g.ExitAnimating {
			g.ExitAnimating = true
			g.ExitAnimStartTime = time.Now().UnixMilli()
		}
	}

	gameplay.PickUpItemsOnFloor(g)
	gameplay.CheckAdjacentGenerators(g)
	gameplay.UpdateLightingExploration(g)

	g.RemoveOldMessages()

	gameplay.ShowInteractableHints(g)
	gameplay.ShowMovementHint(g)

	renderer.RenderFrame(g)

	// If exit animation is running, continue loop without waiting for input
	// This allows the animation to complete automatically
	if g.ExitAnimating {
		// Small delay to allow animation to render smoothly
		time.Sleep(16 * time.Millisecond) // ~60 FPS
		return
	}

	// Get and process input (tiered input system -> Intent -> game logic)
	gameplay.ProcessIntent(g, renderer.Current.GetInput())
	if gameplay.IsLongUseActive(g) {
		gameplay.WaitForLongUseComplete(g)
	}
	if gameplay.IsHazardClearActive(g) {
		gameplay.WaitForHazardClearComplete(g)
	}
	if gameplay.IsHazardTourActive(g) {
		gameplay.WaitForHazardTourComplete(g)
	}
}
