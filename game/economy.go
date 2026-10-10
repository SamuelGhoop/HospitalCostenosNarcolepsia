package game

import "time"

// reviewFee es lo que paga un episodio en la revisión del médico (GAME_DESIGN
// §5.8), según cuánto esperó el paciente desde el desplome: reviewFeeMax
// hasta reviewFeeGrace; después baja reviewFeeDrop por cada segundo
// completo, sin bajar de reviewFeeMin.
//
// La división entera de time.Duration cuenta solo segundos completos: a los
// 15,9 s todavía paga lo máximo.
func reviewFee(waited time.Duration) int {
	if waited <= reviewFeeGrace {
		return reviewFeeMax
	}
	late := int((waited - reviewFeeGrace) / time.Second) // segundos completos de más
	return max(reviewFeeMax-reviewFeeDrop*late, reviewFeeMin)
}
