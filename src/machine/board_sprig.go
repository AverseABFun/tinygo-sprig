//go:build sprig

package machine

import (
	"time"

	"slices"
)

var (
	left_led_pwm   *pwmGroup
	LEFT_WHITE_LED uint8 // This value is the PWM channel ID for the white LED on the sprig. You usually will not want to use this directly, instead using TurnOnWhiteLED or TurnOffWhiteLED.
	right_led_pwm  *pwmGroup
	RIGHT_BLUE_LED uint8 // This value is the PWM channel ID for the blue LED on the sprig. You usually will not want to use this directly, instead using TurnOnBlueLED or TurnOffBlueLED.
)

var (
	LEFT_WHITE_LED_DUTY_CYCLE uint32 = 8 // This value is the divisor for the PWM group's Top value. This means that if you want it on 1/4 of the time, you set this to 4.
	RIGHT_BLUE_LED_DUTY_CYCLE uint32 = 4 // See the note on LEFT_WHITE_LED_DUTY_CYCLE
)

const HISTORY_LEN = 64

type ButtonState struct {
	History    [HISTORY_LEN]bool
	LastState  bool
	ReadState  bool
	HistoryIdx uint8
	Pin        Pin
}

// AddStateFromPin appends a state to History from the set Pin.
//
// This function gets the current value from state.Pin and
// runs state.AddState with it's value. It then returns the value
// that AddState returns.
func (state *ButtonState) AddStateFromPin() bool {
	return state.AddState(state.Pin.Get())
}

// AddState appends a state to History.
//
// If state.HistoryIdx is over or equal to HISTORY_LEN, then it returns false.
// Otherwise, it sets state.LastState to the provide item and adds the state to
// state.History at state.HistoryIdx. After this, it increments state.HistoryIdx and returns true.
func (state *ButtonState) AddState(item bool) bool {
	if state.HistoryIdx >= HISTORY_LEN {
		return false
	}
	state.LastState = item
	state.History[state.HistoryIdx] = item
	state.HistoryIdx++
	return true
}

// ClearHistory clears the History of state.
//
// If state.HistoryIdx is under HISTORY_LEN, then it will return false as it's first value.
// Otherwise, it will return true as it's first value.
// If the first value is true, then the second value is if the number of values in the history is over 5/6 of them.
// This also sets state.ReadState for other places.
// This is in accordance with spade.
// If the first value is false, then the second value is undefined.
func (state *ButtonState) ClearHistory() (bool, bool) {
	if state.HistoryIdx < HISTORY_LEN {
		return false, false
	}
	state.HistoryIdx = 0
	var number uint8 = 0
	for i := 0; i < HISTORY_LEN; i++ {
		var item = state.History[i]
		if item {
			number++
		}
	}
	state.ReadState = number > (5/6)*HISTORY_LEN
	if number > (5/6)*HISTORY_LEN {
		return true, true
	}
	return true, false
}

const BUTTON_NUM = 8

var BUTTON_PINS = [BUTTON_NUM]Pin{GP5, GP7, GP6, GP8, GP12, GP14, GP13, GP15} // This is a slice of all Pins that are used for buttons, ordered as in BUTTON_MAP. You almost always will not want to use these and will instead want to use BUTTON_STATES.
var BUTTON_STATES = [BUTTON_NUM]ButtonState{}                                 // This is a slice of all ButtonStates, ordered as in BUTTON_MAP. All of these will automatically have their Pin attribute set.

var BUTTON_MAP = map[string]uint{"W": 0, "A": 1, "S": 2, "D": 3, "I": 4, "J": 5, "K": 6, "L": 7} // This is a map from button name(for example, "W") to the index in other arrays(such as BUTTON_PINS and BUTTON_STATES). This can be useful for easily looking up values.

// Takes a single function that is called constantly, right after polling the buttons. It will also clear all buttons history as needed.
func BeginLoop(loop func() bool) {
	for true {
		if !PollAllButtons() {
			ClearAllButtons()
			PollAllButtons()
		}
		if !loop() {
			break
		}
	}
}

func init() {
	var pwm = PWM6
	pwm.Configure(PWMConfig{
		Period: 0, // Automatically picks a good period for LEDs
	})

	ch, err := pwm.Channel(GP28)
	if err != nil {
		println(err.Error())
		return
	}
	LEFT_WHITE_LED = ch
	left_led_pwm = pwm

	pwm = PWM2
	pwm.Configure(PWMConfig{
		Period: 0, // Automatically picks a good period for LEDs
	})

	ch, err = pwm.Channel(GP4)
	if err != nil {
		println(err.Error())
		return
	}

	RIGHT_BLUE_LED = ch
	right_led_pwm = pwm

	for i := 0; i < BUTTON_NUM; i++ {
		pin := BUTTON_PINS[i]
		pin.Configure(PinConfig{
			Mode: PinInputPullup,
		})
		BUTTON_STATES[i] = ButtonState{Pin: pin}
	}
}

// Turns on the white LED to the duty cycle put in LEFT_WHITE_LED_DUTY_CYCLE.
func TurnOnWhiteLED() {
	left_led_pwm.Set(LEFT_WHITE_LED, left_led_pwm.Top()/LEFT_WHITE_LED_DUTY_CYCLE)
}

// Turns off the white LED(sets the duty cycle to 0)
func TurnOffWhiteLED() {
	left_led_pwm.Set(LEFT_WHITE_LED, 0)
}

// Turns on the blue LED to the duty cycle put in RIGHT_BLUE_LED_DUTY_CYCLE.
func TurnOnBlueLED() {
	right_led_pwm.Set(RIGHT_BLUE_LED, left_led_pwm.Top()/RIGHT_BLUE_LED_DUTY_CYCLE)
}

// Turns off the blue LED(sets the duty cycle to 0)
func TurnOffBlueLED() {
	right_led_pwm.Set(RIGHT_BLUE_LED, 0)
}

// Loops through BUTTON_STATES and calls AddStateFromPin on each one.
func PollAllButtons() bool {
	var out = false
	for i := 0; i < BUTTON_NUM; i++ {
		state := BUTTON_STATES[i]
		out = state.AddStateFromPin()
	}
	return out
}

func ClearAllButtons() bool {
	var out = false
	for i := 0; i < BUTTON_NUM; i++ {
		state := BUTTON_STATES[i]
		out, _ = state.ClearHistory()
	}
	return out
}

var (
	SPI_TFT_PORT = SPI0
	SPI_TFT_CS   = GP20
	SPI_TFT_DC   = GP22
	SPI_TFT_RST  = GP26
	SPI_TFT_BL   = GP17
	SPI_TFT_RX   = GP16
	SPI_TFT_TX   = GP19
	SPI_TFT_SCK  = GP18
)

const (
	ST7735_NOP        = 0x00 // No-op
	ST7735_SWRESET    = 0x01 // SoftWare RESET
	ST7735_RDDID      = 0x04 // ReaD Display ID
	ST7735_RDDST      = 0x09 // ReaD Display STatus
	ST7735_RDDPM      = 0x0A // ReaD Display Power
	ST7735_SLPIN      = 0x10 // SLeeP IN
	ST7735_SLPOUT     = 0x11 // SLeeP OUT
	ST7735_PTLON      = 0x12 // ParTiaL mode ON
	ST7735_NORON      = 0x13 // NORmal display mode ON
	ST7735_INVOFF     = 0x20 // display INVersion OFF
	ST7735_INVON      = 0x21 // display INVersion ON
	ST7735_DISPOFF    = 0x28 // DISPlay OFF
	ST7735_DISPON     = 0x29 // DISPlay ON
	ST7735_CASET      = 0x2A // Column Address SET
	ST7735_RASET      = 0x2B // Row Address SET
	ST7735_RAMWR      = 0x2C // RAM WRite
	ST7735_RAMRD      = 0x2E // RAM ReaD
	ST7735_PTLAR      = 0x30 // ParTiaL start/end AddRess set
	ST7735_VSCRDEF    = 0x33
	ST7735_COLMOD     = 0x3A // interface pixel format
	ST7735_MADCTL     = 0x36 // Memory Data access ConTroL
	ST7735_MADCTL_MY  = 0x80
	ST7735_MADCTL_MX  = 0x40
	ST7735_MADCTL_MV  = 0x20
	ST7735_MADCTL_ML  = 0x10
	ST7735_MADCTL_RGB = 0x00
	ST7735_VSCRSADD   = 0x37
	ST7735_FRMCTR1    = 0xB1
	ST7735_FRMCTR2    = 0xB2
	ST7735_FRMCTR3    = 0xB3
	ST7735_INVCTR     = 0xB4
	ST7735_DISSET5    = 0xB6
	ST7735_PWCTR1     = 0xC0
	ST7735_PWCTR2     = 0xC1
	ST7735_PWCTR3     = 0xC2
	ST7735_PWCTR4     = 0xC3
	ST7735_PWCTR5     = 0xC4
	ST7735_VMCTR1     = 0xC5
	ST7735_RDID1      = 0xDA // ReaD ID1
	ST7735_RDID2      = 0xDB // ReaD ID2
	ST7735_RDID3      = 0xDC // ReaD ID3
	ST7735_RDID4      = 0xDD // ReaD ID4
	ST7735_PWCTR6     = 0xFC
	ST7735_GMCTRP1    = 0xE0 // GaMma CorrecTion Positive
	ST7735_GMCTRN1    = 0xE1 // GaMma CorrecTion Negative
)

const (
	ST7735_BLACK   = 0x0000
	ST7735_BLUE    = 0x001F
	ST7735_RED     = 0xF800
	ST7735_GREEN   = 0x07E0
	ST7735_CYAN    = 0x07FF
	ST7735_MAGENTA = 0xF81F
	ST7735_YELLOW  = 0xFFE0
	ST7735_WHITE   = 0xFFFF
)

func TFT_cs_write(val bool) {
	SPI_TFT_CS.Set(val)
}

func TFT_dc_write(val bool) {
	SPI_TFT_DC.Set(val)
}

func TFT_rst_write(val bool) {
	SPI_TFT_RST.Set(val)
}

func TFT_SPI_command(cmd byte) {
	TFT_dc_write(false)
	SPI_TFT_PORT.Transfer(cmd)
}

func TFT_SPI_write_command(cmd byte) {
	TFT_dc_write(false)
	TFT_cs_write(false)
	SPI_TFT_PORT.Transfer(cmd)
	TFT_cs_write(true)
}

func TFT_SPI_data(data ...byte) {
	TFT_dc_write(true)
	SPI_TFT_PORT.Tx(data, nil)
}

func TFT_SPI_write_data(data byte) {
	TFT_dc_write(true)
	TFT_cs_write(false)
	SPI_TFT_PORT.Transfer(data)
	TFT_cs_write(true)
}

func TFT_fill_start(pos_x_start uint16, pos_x_end uint16, pos_y_start uint16, pos_y_end uint16) {
	if !TFT_initialized {
		return
	}
	TFT_cs_write(false)

	TFT_SPI_command(ST7735_CASET)
	TFT_SPI_data(byte((pos_x_start>>8)&(0xFF)), byte((pos_x_start)&(0xFF)))
	TFT_SPI_data(byte((pos_x_end>>8)&(0xFF)), byte((pos_x_end)&(0xFF)))

	TFT_SPI_command(ST7735_RASET)
	TFT_SPI_data(byte((pos_y_start>>8)&(0xFF)), byte((pos_y_start)&(0xFF)))
	TFT_SPI_data(byte((pos_y_end>>8)&(0xFF)), byte((pos_y_end)&(0xFF)))

	TFT_SPI_command(ST7735_RAMWR)

	TFT_dc_write(true)
}

func TFT_fill_send(pixel Color) {
	if !TFT_initialized {
		return
	}
	SPI_TFT_PORT.Transfer(byte((pixel >> 8) & 0xFF))
	SPI_TFT_PORT.Transfer(byte(pixel & 0xFF))
}

func TFT_fill_finish() {
	if !TFT_initialized {
		return
	}
	TFT_cs_write(true)
}

// Hardware resets the TFT display
func TFT_reset() {
	TFT_rst_write(true)
	time.Sleep(time.Millisecond * 10)
	TFT_rst_write(false)
	time.Sleep(time.Millisecond * 10)
	TFT_rst_write(true)
	time.Sleep(time.Millisecond * 10)
}

var TFT_initialized = false

// Initalizes the TFT display
func TFT_init() {
	// Backlight
	{
		SPI_TFT_BL.Configure(PinConfig{
			Mode: PinOutput,
		})
		SPI_TFT_BL.Set(true)
	}

	// init SPI, gpio
	{
		SPI_TFT_PORT.Configure(SPIConfig{
			Frequency: 30000000,
			SDI:       SPI_TFT_RX,
			SDO:       SPI_TFT_TX,
			SCK:       SPI_TFT_SCK,
		})

		SPI_TFT_CS.Configure(PinConfig{
			Mode: PinOutput,
		})
		TFT_cs_write(true)

		SPI_TFT_DC.Configure(PinConfig{
			Mode: PinOutput,
		})
		TFT_dc_write(false)

		SPI_TFT_RST.Configure(PinConfig{
			Mode: PinOutput,
		})
		TFT_rst_write(false)
	}

	TFT_reset()

	TFT_dc_write(false)

	// I don't understand this, but it's from spade soo
	{
		TFT_SPI_write_command(ST7735_SWRESET)
		time.Sleep(time.Millisecond * 150)
		TFT_SPI_write_command(ST7735_SLPOUT)
		time.Sleep(time.Millisecond * 500)
		TFT_SPI_write_command(ST7735_FRMCTR1)
		TFT_SPI_write_data(0x01)
		TFT_SPI_write_data(0x2C)
		TFT_SPI_write_data(0x2D)
		TFT_SPI_write_command(ST7735_FRMCTR2)
		TFT_SPI_write_data(0x01)
		TFT_SPI_write_data(0x2C)
		TFT_SPI_write_data(0x2D)
		TFT_SPI_write_command(ST7735_FRMCTR3)
		TFT_SPI_write_data(0x01)
		TFT_SPI_write_data(0x2C)
		TFT_SPI_write_data(0x2D)

		TFT_SPI_write_data(0x01)
		TFT_SPI_write_data(0x2C)
		TFT_SPI_write_data(0x2D)

		TFT_SPI_write_command(ST7735_INVCTR)
		TFT_SPI_write_data(0x07)
		TFT_SPI_write_command(ST7735_PWCTR1)
		TFT_SPI_write_data(0xA2)
		TFT_SPI_write_data(0x02)
		TFT_SPI_write_data(0x84)
		TFT_SPI_write_command(ST7735_PWCTR2)
		TFT_SPI_write_data(0xC5)
		TFT_SPI_write_command(ST7735_PWCTR3)
		TFT_SPI_write_data(0x0A)
		TFT_SPI_write_data(0x00)
		TFT_SPI_write_command(ST7735_PWCTR4)
		TFT_SPI_write_data(0x8A)
		TFT_SPI_write_data(0x2A)
		TFT_SPI_write_command(ST7735_PWCTR5)
		TFT_SPI_write_data(0x8A)
		TFT_SPI_write_data(0xEE)
		TFT_SPI_write_command(ST7735_VMCTR1)
		TFT_SPI_write_data(0x0E)
		TFT_SPI_write_command(ST7735_INVOFF)
		TFT_SPI_write_command(ST7735_MADCTL)
		TFT_SPI_write_data(0x40 | 0x10 | 0x08)
		TFT_SPI_write_command(ST7735_COLMOD)
		TFT_SPI_write_data(0x05)
	}

	// initializes red version, whatever that means
	{
		TFT_SPI_command(ST7735_CASET)
		TFT_SPI_data(0x00, 0x00, 0x00, 0x7F)

		TFT_SPI_command(ST7735_RASET)
		TFT_SPI_data(0x00, 0x00, 0x00, 0x9F)
	}

	// Gamma correction and final setup
	{
		TFT_SPI_write_command(ST7735_GMCTRP1)
		TFT_SPI_write_data(0x02)
		TFT_SPI_write_data(0x1C)
		TFT_SPI_write_data(0x07)
		TFT_SPI_write_data(0x12)
		TFT_SPI_write_data(0x37)
		TFT_SPI_write_data(0x32)
		TFT_SPI_write_data(0x29)
		TFT_SPI_write_data(0x2D)
		TFT_SPI_write_data(0x29)
		TFT_SPI_write_data(0x25)
		TFT_SPI_write_data(0x2B)
		TFT_SPI_write_data(0x39)
		TFT_SPI_write_data(0x00)
		TFT_SPI_write_data(0x01)
		TFT_SPI_write_data(0x03)
		TFT_SPI_write_data(0x10)
		TFT_SPI_write_command(ST7735_GMCTRN1)
		TFT_SPI_write_data(0x03)
		TFT_SPI_write_data(0x1D)
		TFT_SPI_write_data(0x07)
		TFT_SPI_write_data(0x06)
		TFT_SPI_write_data(0x2E)
		TFT_SPI_write_data(0x2C)
		TFT_SPI_write_data(0x29)
		TFT_SPI_write_data(0x2D)
		TFT_SPI_write_data(0x2E)
		TFT_SPI_write_data(0x2E)
		TFT_SPI_write_data(0x37)
		TFT_SPI_write_data(0x3F)
		TFT_SPI_write_data(0x00)
		TFT_SPI_write_data(0x00)
		TFT_SPI_write_data(0x02)
		TFT_SPI_write_data(0x10)
		TFT_SPI_write_command(ST7735_NORON)
		time.Sleep(time.Millisecond * 10)
		TFT_SPI_write_command(ST7735_DISPON)
		time.Sleep(time.Millisecond * 100)
	}

	TFT_initialized = true
}

// Creates a new Color from the passed values in the range 0-255
func Color16(r uint8, g uint8, b uint8) Color {
	r16 := uint16((float64((float64(r) / 255.0) * 31.0)))
	g16 := uint16((float64((float64(r) / 255.0) * 63.0)))
	b16 := uint16((float64((float64(b) / 255.0) * 31.0)))

	// return ((r & 0xf8) << 8) + ((g & 0xfc) << 3) + (b >> 3);
	return Color(((r16 & 0b11111000) << 8) | ((g16 & 0b11111100) << 3) | (b16 >> 3))
}

type Color uint16

const TEXT_CHARS_MAX_X = 20
const TEXT_CHARS_MAX_Y = 16

const SCREEN_SIZE_X = 160
const SCREEN_SIZE_Y = 128

var (
	text_char            = [TEXT_CHARS_MAX_Y][TEXT_CHARS_MAX_X]byte{}
	text_color           = [TEXT_CHARS_MAX_Y][TEXT_CHARS_MAX_X]Color{}
	text_set             = [TEXT_CHARS_MAX_Y][TEXT_CHARS_MAX_X]bool{}
	Text_x           int = 0
	Text_y           int = 0
	Background_color     = Color16(0, 0, 0)
)

func ClearText() {
	text_char = [TEXT_CHARS_MAX_Y][TEXT_CHARS_MAX_X]byte{}
	text_color = [TEXT_CHARS_MAX_Y][TEXT_CHARS_MAX_X]Color{}
	text_set = [TEXT_CHARS_MAX_Y][TEXT_CHARS_MAX_X]bool{}
	RenderAllText()
}

// Adds a string
// returns if the character would overflow the screen downward while still adding all character up to that point
func AddString(str string, color Color) bool {
	for i := 0; i < len(str); i++ {
		if !AddCharacter(str[i], color) {
			return false
		}
	}
	return true
}

// Adds a character and increments the proper variables
// returns if the character would overflow the screen downward
func AddCharacter(char byte, color Color) bool {
	if char == "\n"[0] {
		Text_y++
		Text_x = 0
		return Text_y <= TEXT_CHARS_MAX_Y
	}
	Text_x++
	if Text_x > TEXT_CHARS_MAX_X {
		Text_x = 0
		Text_y++
	}
	if Text_y > TEXT_CHARS_MAX_Y {
		return false
	}
	SetCharacter(char, color, Text_x, Text_y)
	return true
}

// Sets the character at position (x, y) to character char and color color
func SetCharacter(char byte, color Color, x int, y int) {
	if !TFT_initialized {
		return
	}
	text_char[y][x] = char
	text_color[y][x] = color
	text_set[y][x] = true
	RenderChar(x, y)
}

func RenderChar(x int, y int) {
	if !TFT_initialized {
		return
	}
	var char = TFT_font[text_char[y][x] : text_char[y][x]+8]
	var real_x, real_y = x * 8, y * 8
	TFT_fill_start(uint16(real_x), uint16(real_x)+8, uint16(real_y), uint16(real_y)+8)
	for _, val := range char {
		for i := 0; i < 8; i++ {
			val2 := val >> i
			val2 = val2 & 0x1
			if val2 == 1 && text_set[y][x] {
				TFT_fill_send(text_color[y][x])
			} else {
				TFT_fill_send(Background_color)
			}
		}
	}
	TFT_fill_finish()
}

func RenderAllText() {
	if !TFT_initialized {
		return
	}
	for y := 0; y < TEXT_CHARS_MAX_Y; y++ {
		for x := 0; x < TEXT_CHARS_MAX_X; x++ {
			RenderChar(x, y)
		}
	}
}

type SpriteList []*Sprite

func (spriteList *SpriteList) Register(sprite *Sprite) {
	*spriteList = SpriteList(append(*spriteList, sprite))
	sprite.registered = append(sprite.registered, spriteList)
}

func (spriteList SpriteList) Render() {
	var mapOf = map[Position][]*Sprite{}
	for _, sprite := range spriteList {
		mapOf[CreatePosition(sprite.X, sprite.Y)] = append(mapOf[CreatePosition(sprite.X, sprite.Y)], sprite)
	}
	for _, sprites := range mapOf {
		slices.SortFunc(sprites, func(a *Sprite, b *Sprite) int {
			return int(a.Z - b.Z)
		})
		var spriteData map[Position]Color
		for _, sprite := range sprites {
			for y, val := range sprite.Data {
				for x, color := range val {
					spriteData[CreatePosition(uint8(x), uint8(y))] = color
				}
			}
		}
		for _, sprite := range sprites {
			sprite.render(spriteData)
		}
	}
}

func CreatePosition(x uint8, y uint8) Position {
	return Position(y | (x << 8))
}

type Position uint16

func (pos Position) GetX() uint8 {
	return uint8((pos >> 8) & (0xFF))
}

func (pos Position) GetY() uint8 {
	return uint8((pos) & (0xFF))
}

type Sprite struct {
	X          uint8
	Y          uint8
	Z          uint8
	Size_X     uint8
	Size_Y     uint8
	Data       [][]Color
	registered []*SpriteList
}

func (sprite Sprite) render(custom_pixel_data map[Position]Color) {
	TFT_fill_start(uint16(sprite.X), uint16(sprite.X+sprite.Size_X), uint16(sprite.Y), uint16(sprite.Y+sprite.Size_Y))
	for y, val := range sprite.Data {
		for x, clr := range val {
			var writing_sprite_pixel = true
			for pos, color := range custom_pixel_data {
				if pos.GetX() == uint8(x) && pos.GetY() == uint8(y) && clr == Background_color {
					writing_sprite_pixel = false
					TFT_fill_send(color)
				}
			}
			if writing_sprite_pixel {
				TFT_fill_send(clr)
			}
		}
	}
	TFT_fill_finish()
}
