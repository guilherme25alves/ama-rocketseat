interface GetRoomMessageRequest {
    roomId: string
}

export async function getRoomMessages({ roomId } : GetRoomMessageRequest) {
    const response = await fetch(`${import.meta.env.VITE_APP_API_URL}/rooms/${roomId}/messages`)

    const data: Array<{
        id: string
        room_id: string
        message: string
        reaction_count: number
        answered: boolean
    }> = await response.json()

    return data != null ? {
        messages: data.map(item => {
            return {
                id: item.id,
                text: item.message,
                amountOfReactions: item.reaction_count,
                answered: item.answered
            }
        })
    } : []
}